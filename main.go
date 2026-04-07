package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gookit/goutil/cflag"
	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/handlers"
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

func main() {
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

	cmd.IntVar(&config.Cfg.PortInt, "port", 0, "HTTP port to listen on;;p")
	cmd.Func = run
	cmd.QuickRun()
}

func run(c *cflag.CFlags) error {
	args := c.RemainArgs()

	// - Prepare arguments
	prepare(args)

	fmt.Printf("Serving directory: %s\n", config.Cfg.TargetDir)
	fmt.Printf("Default entry file: %s\n", config.Cfg.EntryFile)
	fmt.Printf("🚀 Server running at http://localhost:%s\n", config.Cfg.PortStr())

	// - Watcher
	if config.Cfg.EnableWatch {
		go handlers.WatchDirectory(config.Cfg.TargetDir)
	}

	log.Fatal(http.ListenAndServe(":"+config.Cfg.PortStr(), newServerMux()))
	return nil
}

func newServerMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/static/", newStaticHandler())
	mux.HandleFunc("/sse", handlers.HandleSSE)
	mux.HandleFunc("/api/file-tree", handlers.HandleFileTreeAPI)
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
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		distHandler.ServeHTTP(w, r)
	})
}

func prepare(args []string) {
	err := envutil.DotenvLoad(func(cfg *envutil.Dotenv) {
		cfg.IgnoreNotExist = true
	})
	if err != nil {
		clog.Warnf("Failed to load dotenv: %v", err)
	}

	cwd, _ := os.Getwd()
	targetDir := cwd
	if len(args) > 0 {
		absPath, err := filepath.Abs(args[0])
		if err == nil {
			targetDir = absPath
		} else {
			clog.Warnf("Failed to resolve absolute path: %s, err: %v", args[0], err)
		}
	}

	var entryFile string
	if len(args) > 1 && args[1] != "" {
		entryFile = args[1]
	}

	utils.EnableDebug = envutil.GetBool(config.EnvDebug, false)
	config.EnableDebug = utils.EnableDebug
	config.Cfg.Init(targetDir, entryFile)

	handlers.IfsReader = func(path string) ([]byte, error) {
		return content.ReadFile(path)
	}
}
