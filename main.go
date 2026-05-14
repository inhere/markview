package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/goutil/cflag"
	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/netutil"
	"github.com/gookit/goutil/sysutil"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/handlers"
	"github.com/inhere/markview/internal/projects"
	"github.com/inhere/markview/internal/utils"
)

//go:embed web/template.html web/template-main.html web/dist
var content embed.FS

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	GitHash   = "unknown"
	BuildTime = "unknown"
)

var openBrowser = sysutil.OpenBrowser

func main() {
	cmd := newCommand()
	cmd.QuickRun()
}

func newCommand() *cflag.CFlags {
	// 1. Configuration
	clog.Configure(func(p *clog.Printer) {
		p.TimeFormat = "15:04:05"
		p.Template = "{time} | {emoji} {message}"
	})

	cmd := cflag.New()
	cmd.Desc = fmt.Sprintf(
		"MarkView - Markdown Live Preview Server\n (Version: %s, Git Hash: %s, Build Time: %s)",
		Version, GitHash, BuildTime,
	)
	cmd.LongHelp = fmt.Sprintf(`
<cyan>Arguments:</>
  directory      Directory to watch (default: current dir)
  default-entry  Default markdown file to open (default: %s)

<cyan>Environment:</>
  MKVIEW_PORT    HTTP port to listen on (default: %s)
  MKVIEW_ENTRY   Default markdown file to open (default: %s)`,
		config.DefaultEntry, config.DefaultPort, config.DefaultEntry,
	)

	cmd.IntVar(&config.Cfg.PortInt, "port", 0,
		"HTTP port to listen on, if < 0, use random port;;p",
	)
	cmd.BoolVar(&config.Cfg.Private, "private", false,
		"Only listen on localhost (127.0.0.1), not publicly accessible",
	)
	cmd.BoolVar(&config.Cfg.NoBrowser, "no-browser", false,
		"Do not open the local preview URL in browser after server starts",
	)
	cmd.StringVar(&projectsAction, "projects", "",
		"Manage saved projects: list, show, remove, prune",
	)
	cmd.Func = run
	return cmd
}

func run(c *cflag.CFlags) error {
	markPortFlagVisited(c)
	args := c.RemainArgs()
	if projectsAction != "" {
		return runProjectsAction(projectsAction, args, os.Stdout)
	}

	// Prepare arguments
	if err := prepare(args); err != nil {
		return err
	}

	fmt.Printf("Serving directory: %s\n", config.Cfg.TargetDir)
	fmt.Printf("Default entry file: %s\n", config.Cfg.EntryFile)

	// - Watcher
	if config.Cfg.EnableWatch {
		go handlers.WatchDirectory(config.Cfg.TargetDir)
	}

	// 创建 SSE 端点 URL，供前端使用
	sseURL := "/sse"

	// 主 server：处理静态页面、API 等
	// 问题根因：原全局 WriteTimeout: 10s 与 SSE keepalive(9s) 冲突，导致 18s 时断连
	// 修复方案：
	//   1. 移除 server 级 WriteTimeout
	//   2. API 路由通过 http.TimeoutHandler 在 handler 级别控制超时
	//   3. SSE 路由不受超时限制，保持长连接
	mainServer := &http.Server{
		Addr:        config.Cfg.ListenAddr(),
		Handler:     newServerMux(sseURL),
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 120 * time.Second,
		// 注意：不设置 WriteTimeout，让 SSE 长连接不受限制
		// API 超时通过 http.TimeoutHandler 在路由级别处理
	}

	if shouldUseProjectPortRegistry() {
		listener, actualPort, err := listenAndRememberProjectPort(config.Cfg.TargetDir)
		if err != nil {
			log.Fatal("Failed to create listener:", err)
		}

		config.Cfg.SetPort(actualPort)
		beforeServerRun(actualPort, config.Cfg.Private)
		log.Fatal(mainServer.Serve(listener))
		return nil
	}

	// 启动服务器并获取实际端口（支持随机端口）
	isRandomPort := config.Cfg.PortInt < 0
	if isRandomPort {
		// 随机端口模式：创建监听器获取实际端口
		listener, err := net.Listen("tcp", mainServer.Addr)
		if err != nil {
			log.Fatal("Failed to create listener:", err)
		}

		actualPort := listener.Addr().(*net.TCPAddr).Port
		config.Cfg.SetPort(actualPort)
		beforeServerRun(actualPort, config.Cfg.Private)
		log.Fatal(mainServer.Serve(listener))
		return nil
	}

	beforeServerRun(config.Cfg.PortInt, config.Cfg.Private)
	log.Fatal(mainServer.ListenAndServe())
	return nil
}

func markPortFlagVisited(c *cflag.CFlags) {
	c.Visit(func(flag *flag.Flag) {
		if flag.Name == "port" {
			config.Cfg.PortSource = config.PortSourceCLI
		}
	})
}

func shouldUseProjectPortRegistry() bool {
	// 只有自动端口场景使用项目端口记忆，显式固定端口和 ENV 端口保持完全可预期。
	return config.Cfg.PortSource == config.PortSourceUnset ||
		(config.Cfg.PortSource == config.PortSourceCLI && config.Cfg.PortInt < 0)
}

func listenAndRememberProjectPort(targetDir string) (net.Listener, int, error) {
	registryPath, err := projects.RegistryPath()
	if err != nil {
		clog.Warnf("Failed to resolve project registry path: %v", err)
	}

	registry := projects.Registry{}
	if registryPath != "" {
		loaded, err := projects.Load(registryPath)
		if err != nil {
			clog.Warnf("Failed to load project registry, starting with empty registry: %v", err)
		} else {
			registry = loaded
		}
	}

	host := ""
	if config.Cfg.Private {
		host = "127.0.0.1"
	}
	preferDefault := config.Cfg.PortSource == config.PortSourceUnset
	listener, actualPort, err := listenProjectPortFromRegistry(host, targetDir, registry, preferDefault)
	if err != nil {
		return nil, 0, err
	}

	if err := projects.Upsert(registry, targetDir, actualPort, time.Now()); err != nil {
		clog.Warnf("Failed to update project registry: %v", err)
		return listener, actualPort, nil
	}
	if registryPath != "" {
		if err := projects.Save(registryPath, registry); err != nil {
			clog.Warnf("Failed to save project registry: %v", err)
		}
	}

	return listener, actualPort, nil
}

func listenProjectPortFromRegistry(host string, targetDir string, registry projects.Registry, preferDefault bool) (net.Listener, int, error) {
	if savedPort, ok := projects.LookupPort(registry, targetDir); ok {
		// 已保存端口优先；若被占用，从该端口继续向后找可用端口并保存新结果。
		if listener, port, err := listenNextAvailable(host, savedPort, 100); err == nil {
			return listener, port, nil
		}
	}

	if preferDefault {
		if defaultPort, err := strconv.Atoi(config.DefaultPort); err == nil {
			// 未设置端口的默认启动仍优先使用 6100，保证老用户的默认 URL 不变化。
			if listener, port, err := listenNextAvailable(host, defaultPort, 100); err == nil {
				return listener, port, nil
			}
		}
	}

	// 兜底交给系统随机端口，避免缓存文件或连续端口占用阻止服务启动。
	listener, err := net.Listen("tcp", net.JoinHostPort(normalizeListenHost(host), "0"))
	if err != nil {
		return nil, 0, err
	}
	return listener, listener.Addr().(*net.TCPAddr).Port, nil
}

func listenNextAvailable(host string, startPort int, limit int) (net.Listener, int, error) {
	host = normalizeListenHost(host)
	for port := startPort; port < startPort+limit; port++ {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
		if err == nil {
			return listener, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no available port found from %d to %d", startPort, startPort+limit-1)
}

func normalizeListenHost(host string) string {
	if host == "" {
		return "0.0.0.0"
	}
	return host
}

func beforeServerRun(port int, private bool) {
	localUrl := fmt.Sprintf("http://127.0.0.1:%d", port)
	if private {
		fmt.Printf("🚀 Live server running at %s (PRIVATE MODE)\n", localUrl)
		openLocalPreview(localUrl)
		return
	}

	fmt.Printf("🚀 Live server running at %s\n", localUrl)
	openLocalPreview(localUrl)
	ips, err := netutil.AllLocalIPv4()
	if err != nil {
		clog.Info("Failed to get local IPs:", err)
		return
	}
	if len(ips) == 0 {
		return
	}

	// 打印所有 IP 地址的访问 URL
	urls := []string{}
	for _, ip := range ips {
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
	}
	fmt.Printf(" - Can also access by %s\n", strings.Join(urls, ", "))
}

func openLocalPreview(localUrl string) {
	if config.Cfg.NoBrowser {
		return
	}

	// 打开浏览器失败不应影响服务启动，只记录告警方便排查本机环境问题。
	if err := openBrowser(localUrl); err != nil {
		clog.Warnf("Failed to open browser: %v", err)
	}
}

// newServerMux 创建路由 mux，SSE 路由单独处理
func newServerMux(ssePath string) *http.ServeMux {
	mux := http.NewServeMux()

	// 静态文件处理
	mux.Handle("/static/", newStaticHandler())

	// SSE 路由：不走超时限制，支持长连接
	mux.HandleFunc(ssePath, handlers.HandleSSE)

	// API 路由：使用 TimeoutHandler 限制最大处理时间（10秒），防止慢连接攻击
	// 与原来的 WriteTimeout: 10s 效果相同，但只在 handler 级别生效
	apiHandler := http.TimeoutHandler(
		http.HandlerFunc(handlers.HandleSearch),
		10*time.Second, "request timeout",
	)
	mux.HandleFunc("/api/search", apiHandler.ServeHTTP)

	fileTreeHandler := http.TimeoutHandler(
		http.HandlerFunc(handlers.HandleFileTreeAPI),
		10*time.Second, "request timeout",
	)
	mux.HandleFunc("/api/file-tree", fileTreeHandler.ServeHTTP)

	// 主页面处理
	mux.HandleFunc("/", handlers.HandleRequest)
	return mux
}

func newStaticHandler() http.Handler {
	distFS, err := fs.Sub(content, "web/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}

	distHandler := http.StripPrefix("/static/", http.FileServer(http.FS(distFS)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.Debugf("Request: %s handle static file", r.URL.Path)
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		distHandler.ServeHTTP(w, r)
	})
}

func prepare(args []string) error {
	err := envutil.DotenvLoad(func(cfg *envutil.Dotenv) {
		cfg.IgnoreNotExist = true
	})
	if err != nil {
		clog.Warnf("Failed to load dotenv: %v", err)
	}

	var entryFile string
	cwd, _ := os.Getwd()
	targetDir := cwd

	// Check target directory
	if len(args) > 0 {
		firstArg := args[0]
		absPath, err := filepath.Abs(firstArg)
		if err == nil {
			// up: 如果第一个参数是 md 文件，作为入口文件
			if strings.HasSuffix(firstArg, ".md") && fsutil.IsFile(firstArg) {
				entryFile = firstArg
			} else {
				targetDir = absPath
			}
		} else {
			clog.Warnf("Failed to resolve absolute path: %s, err: %v", firstArg, err)
		}
	}

	// Check entry file
	if len(args) > 1 && args[1] != "" && entryFile == "" {
		entryFile = args[1]
	}

	utils.EnableDebug = envutil.GetBool(config.EnvDebug, false)
	config.EnableDebug = utils.EnableDebug
	if err := config.Cfg.Init(targetDir, entryFile); err != nil {
		return err
	}

	handlers.IfsReader = func(path string) ([]byte, error) {
		return content.ReadFile(path)
	}
	return nil
}
