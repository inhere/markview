package bootstrap

import (
	"context"
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
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/handlers"
	"github.com/inhere/markview/internal/projects"
	"github.com/inhere/markview/internal/utils"
)

type options struct {
	Content   fs.FS
	Version   string
	GitHash   string
	BuildTime string
}

var appOptions options
var showVersion bool
var globalMode bool
var openBrowser = sysutil.OpenBrowser
var cliPortFlagVisited bool
var cliPrivateFlagVisited bool

// Run 启动 CLI 入口，main 包只负责传入嵌入资源和构建信息。
func Run(content fs.FS, version string, gitHash string, buildTime string) {
	cmd := newCommand(options{
		Content:   content,
		Version:   version,
		GitHash:   gitHash,
		BuildTime: buildTime,
	})
	cmd.QuickRun()
}

func newCommand(options options) *cflag.CFlags {
	appOptions = options // save global options
	// 初始化日志格式集中在 bootstrap，main 只负责传入资源和版本信息。
	clog.Configure(func(p *clog.Printer) {
		p.TimeFormat = "15:04:05"
		p.Template = "{time} | {emoji} {message}"
	})

	cmd := cflag.New()
	cmd.Desc = fmt.Sprintf(
		"MarkView - Markdown Live Preview Server (Version: %s, Build At: %s)",
		options.Version, options.BuildTime,
	)
	cmd.LongHelp = fmt.Sprintf(`
<cyan>Arguments:</>
  directory      Directory to watch (default: current dir)
  default-entry  Default markdown file to open (default: %s)

<cyan>Environment:</>
  MKVIEW_PORT            HTTP port to listen on (unset: auto, prefers %s)
  MKVIEW_ENTRY           Default markdown file to open (default: %s)
  MKVIEW_DEBUG           Enable debug logs (true/false, default: false)
  MKVIEW_WATCH           Watch files and live reload (true/false, default: true)
  MKVIEW_WATCH_DIR       Directories to watch, comma separated (default: target dir)
  MKVIEW_WATCH_SKIP_DIR  Directories to skip while watching; supports append:/override:
  MKVIEW_INCLUDE_DIR     Skipped directories to include in file tree, comma separated
  MKVIEW_PREVIEW_EXTS    Preview extensions; supports append:/override:
  MKVIEW_IFRAME_HOSTS    External iframe host allowlist, comma separated`,
		config.DefaultEntry, config.DefaultPort, config.DefaultEntry,
	)

	cmd.IntVar(&config.Cfg.PortInt, "port", 0,
		"HTTP port to listen on;;p",
	)
	cmd.BoolVar(&config.Cfg.Private, "private", false,
		"Only listen on localhost (127.0.0.1), not publicly accessible",
	)
	cmd.BoolVar(&config.Cfg.NoBrowser, "no-browser", false,
		"Do not open the local preview URL in browser after server starts",
	)
	cmd.BoolVar(&globalMode, "global", false,
		"Serve all saved projects from one local server",
	)
	cmd.StringVar(&projectsAction, "projects", "",
		"Manage saved projects: list, show, remove, prune;;ps",
	)
	cmd.StringVar(&selectedProject, "project", "",
		"Start a saved project by name or path;;proj,P",
	)
	cmd.BoolVar(&showVersion, "version", false, "Show version information;;V")
	cmd.Func = func(c *cflag.CFlags) error {
		return run(c, options.Content)
	}
	cmd.AfterFlagParse = func(c *cflag.CFlags) bool {
		if showVersion {
			fmt.Printf("%s (%s, %s)", options.Version, options.GitHash, options.BuildTime)
			return false
		}
		return true
	}
	return cmd
}

func run(c *cflag.CFlags, content fs.FS) error {
	markCliFlagVisits(c)
	args := c.RemainArgs()
	if globalMode {
		if err := validateGlobalMode(args); err != nil {
			return err
		}
		return runGlobal(content)
	}
	if projectsAction != "" {
		return runProjectsAction(projectsAction, args, os.Stdout)
	}
	if selectedProject != "" {
		targetDir, err := resolveSelectedProjectTarget(selectedProject)
		if err != nil {
			return err
		}
		args, err = buildPrepareArgsForSelectedProject(targetDir, args)
		if err != nil {
			return err
		}
	}

	if err := prepare(args, content); err != nil {
		return err
	}

	ccolor.Printf("Serving directory: <info>%s</>\n", config.Cfg.TargetDir)
	ccolor.Printf("Default entry file: <info>%s</>\n", config.Cfg.EntryFile)

	projectRoot, err := handlers.NewProjectRoot(config.Cfg.TargetDir)
	if err != nil {
		return err
	}
	events := handlers.NewEventHub()
	projectServer := handlers.NewProjectServer(config.Cfg, projectRoot, events, content)
	if config.Cfg.EnableWatch {
		projectWatcher, err := handlers.NewWatcher(projectRoot, config.Cfg, events)
		if err != nil {
			return err
		}
		go func() {
			if err := projectWatcher.Run(context.Background()); err != nil {
				clog.Errorf("WATCH: %v", err)
			}
		}()
	}

	// SSE 是长连接，server 级别不设置 WriteTimeout；普通 API 通过 TimeoutHandler 限时。
	mainServer := &http.Server{
		Addr:        config.Cfg.ListenAddr(),
		Handler:     newServerMux(content, projectServer),
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 120 * time.Second,
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

	beforeServerRun(config.Cfg.PortInt, config.Cfg.Private)
	log.Fatal(mainServer.ListenAndServe())
	return nil
}

func buildPrepareArgsForSelectedProject(targetDir string, args []string) ([]string, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("--project accepts at most one entry file argument")
	}
	if len(args) == 1 && args[0] != "" {
		return []string{targetDir, args[0]}, nil
	}
	return []string{targetDir}, nil
}

func markPortFlagVisited(c *cflag.CFlags) {
	markCliFlagVisits(c)
}

func markCliFlagVisits(c *cflag.CFlags) {
	cliPortFlagVisited = false
	cliPrivateFlagVisited = false
	c.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "port":
			// PortSource 是 prepare 后的运行态；单独记录本次解析，避免下一次 CLI 复用旧来源。
			cliPortFlagVisited = true
			config.Cfg.PortSource = config.PortSourceCLI
		case "private":
			// Bool flags can be explicitly set false; track visitation separately from the value.
			cliPrivateFlagVisited = true
		}
	})
}

func shouldUseProjectPortRegistry() bool {
	// 只有自动端口场景使用项目端口记忆，显式固定端口和 ENV 端口保持完全可预期。
	return config.Cfg.PortSource == config.PortSourceUnset ||
		config.Cfg.PortSource == config.PortSourceRegistry
}

func listenAndRememberProjectPort(targetDir string) (net.Listener, int, error) {
	registryPath, err := projectRegistryPath()
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
	reservedPorts := reservedProjectPorts(registry, targetDir, false)
	if savedPort, ok := projects.LookupPort(registry, targetDir); ok {
		// 已保存端口优先；若其他项目也记录了该端口或端口被占用，则继续向后找可用端口。
		if listener, port, err := listenNextAvailable(host, savedPort, 100, reservedPorts); err == nil {
			return listener, port, nil
		}
	}

	if preferDefault {
		if defaultPort, err := strconv.Atoi(config.DefaultPort); err == nil {
			// 未设置端口的默认启动仍优先使用 6100，保证老用户的默认 URL 不变化。
			if listener, port, err := listenNextAvailable(host, defaultPort, 100, reservedPorts); err == nil {
				return listener, port, nil
			}
		}
	}

	// 兜底交给系统随机端口，避免缓存文件或连续端口占用阻止服务启动。
	return listenRandomPort(host, reservedPorts)
}

func reservedProjectPorts(registry projects.Registry, targetDir string, includeCurrentProject bool) map[int]struct{} {
	key, err := projects.ProjectKey(targetDir)
	if err != nil {
		return nil
	}

	reserved := make(map[int]struct{})
	for path, record := range registry {
		if (!includeCurrentProject && path == key) || record.Port <= 0 {
			continue
		}
		reserved[record.Port] = struct{}{}
	}
	return reserved
}

func listenNextAvailable(host string, startPort int, limit int, reservedPorts map[int]struct{}) (net.Listener, int, error) {
	host = normalizeListenHost(host)
	for port := startPort; port < startPort+limit; port++ {
		if _, reserved := reservedPorts[port]; reserved {
			continue
		}
		listener, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
		if err == nil {
			return listener, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no available port found from %d to %d", startPort, startPort+limit-1)
}

func listenRandomPort(host string, reservedPorts map[int]struct{}) (net.Listener, int, error) {
	host = normalizeListenHost(host)
	for attempt := 0; attempt < 100; attempt++ {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			return nil, 0, err
		}
		port := listener.Addr().(*net.TCPAddr).Port
		if _, reserved := reservedPorts[port]; !reserved {
			return listener, port, nil
		}
		_ = listener.Close()
	}
	return nil, 0, fmt.Errorf("no random port available outside project registry reservations")
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

func newServerMux(content fs.FS, project http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/static/", newStaticHandler(content))
	mux.Handle("/", project)
	return mux
}

func newStaticHandler(content fs.FS) http.Handler {
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

func prepare(args []string, content fs.FS) error {
	targetDir, entryFile := resolvePrepareTarget(args)
	dotenv, err := loadProjectDotenv(targetDir)
	if err != nil {
		clog.Warnf("Failed to load dotenv: %v", err)
	}

	merged, err := buildRuntimeConfig(targetDir, dotenv)
	if err != nil {
		return err
	}
	config.Cfg = merged
	config.Cfg.Version = appOptions.Version

	utils.EnableDebug = runtimeDebugEnabled(dotenv)
	config.EnableDebug = utils.EnableDebug
	if err := config.Cfg.Init(targetDir, entryFile); err != nil {
		return err
	}

	handlers.IfsReader = func(path string) ([]byte, error) {
		return fs.ReadFile(content, path)
	}
	return nil
}

func resolvePrepareTarget(args []string) (string, string) {
	var entryFile string
	cwd, _ := os.Getwd()
	targetDir := cwd

	if len(args) > 0 {
		firstArg := args[0]
		absPath, err := filepath.Abs(firstArg)
		if err == nil {
			if strings.HasSuffix(firstArg, ".md") && fsutil.IsFile(firstArg) {
				entryFile = firstArg
			} else {
				targetDir = absPath
			}
		} else {
			clog.Warnf("Failed to resolve absolute path: %s, err: %v", firstArg, err)
		}
	}

	if len(args) > 1 && args[1] != "" && entryFile == "" {
		entryFile = args[1]
	}

	return targetDir, entryFile
}

func loadProjectDotenv(targetDir string) (map[string]string, error) {
	return config.LoadProjectDotenv(targetDir)
}

func buildRuntimeConfig(targetDir string, dotenv map[string]string) (config.Config, error) {
	registryPort := lookupProjectRegistryPort(targetDir)
	cliPort := (*int)(nil)
	if cliPortFlagVisited {
		cliPort = &config.Cfg.PortInt
	}
	cliPrivate := (*bool)(nil)
	if cliPrivateFlagVisited {
		cliPrivate = &config.Cfg.Private
	}

	return config.LoadProjectRuntimeConfig(targetDir, config.ProjectLoadOptions{
		RegistryPort: registryPort,
		CLI: config.CLIConfig{
			Port:    cliPort,
			Private: cliPrivate,
		},
		NoBrowser: config.Cfg.NoBrowser,
	})
}

func lookupProjectRegistryPort(targetDir string) *int {
	registryPath, err := projectRegistryPath()
	if err != nil {
		clog.Warnf("Failed to resolve project registry path: %v", err)
		return nil
	}
	registry, err := projects.Load(registryPath)
	if err != nil {
		clog.Warnf("Failed to load project registry, ignoring saved port: %v", err)
		return nil
	}
	port, ok := projects.LookupPort(registry, targetDir)
	if !ok {
		return nil
	}
	return &port
}

func runtimeEnvConfig(dotenv map[string]string) (config.EnvConfig, error) {
	return config.RuntimeEnvConfig(dotenv)
}

func envValue(name string, dotenv map[string]string) string {
	if dotenv != nil {
		if value, ok := dotenv[name]; ok {
			return value
		}
	}
	return envutil.Getenv(name, "")
}

func runtimeDebugEnabled(dotenv map[string]string) bool {
	raw := envValue(config.EnvDebug, dotenv)
	if raw == "" {
		return false
	}
	value, err := strconv.ParseBool(raw)
	return err == nil && value
}
