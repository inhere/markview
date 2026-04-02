package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/goutil/cflag"
	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/x/clog"
)

//go:embed web/template.html web/template-main.html web/dist
var content embed.FS

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	GitHash   = "unknown"
	BuildTime = "unknown"
)

var (
	targetDir    string
	defaultEntry string
	portInt      int
	port         string
	enableDebug   bool
)

var (
	enableWatch   bool
	watchDirs     []string
	watchSkipDirs []string
)
var defaultSkipDirs = []string{
	"node_modules",
	"dist",
	"tmp",
	"temp",
}

var (
	clients   = make(map[chan string]bool)
	clientsMu sync.Mutex
	watcher   *fsnotify.Watcher
)

const (
	DefaultPort  = "6100"
	DefaultEntry = "README.md"
)

const (
	EnvPort  = "MKVIEW_PORT"
	EnvEntry = "MKVIEW_ENTRY"
	EnvDebug = "MKVIEW_DEBUG"
	EnvWatch = "MKVIEW_WATCH"
	// Watch directory. multi use comma split
	EnvWatchDir = "MKVIEW_WATCH_DIR"
	// Watch skip directory. multi use comma split
	//  - 前缀 override: 覆盖默认的设置, append: 追加到默认的设置
	EnvWatchSkipDir = "MKVIEW_WATCH_SKIP_DIR"
)

type PageData struct {
	Title               string
	Content             template.HTML
	MainContent         template.HTML
	FileName            string
	FilePath            string
	FileSize            string
	CreatedAt           string
	ModifiedAt          string
	FileTreeJSON        template.JS
	CurrentFilePathJSON template.JS
}

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
		DefaultEntry, DefaultPort, DefaultEntry,
	)

	cmd.IntVar(&portInt, "port", 0, "HTTP port to listen on;;p")
	cmd.Func = run
	cmd.QuickRun()
}

func run(c *cflag.CFlags) error {
	args := c.RemainArgs()

	// - Prepare arguments
	prepare(args)

	fmt.Printf("Serving directory: %s\n", targetDir)
	fmt.Printf("Default entry file: %s\n", defaultEntry)
	fmt.Printf("🚀 Server running at http://localhost:%s\n", port)

	// - Watcher
	if enableWatch {
		go watchDirectory(targetDir)
	}

	log.Fatal(http.ListenAndServe(":"+port, newServerMux()))
	return nil
}

func newServerMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/static/", newStaticHandler())
	mux.HandleFunc("/sse", handleSSE)
	mux.HandleFunc("/", handleRequest)
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
		setStaticCacheHeaders(w)
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
	targetDir = cwd
	if len(args) > 0 {
		absPath, err := filepath.Abs(args[0])
		if err == nil {
			targetDir = absPath
		}
	}

	if len(args) > 1 && args[1] != "" {
		defaultEntry = args[1]
	} else {
		defaultEntry = envutil.Getenv(EnvEntry, DefaultEntry)
	}

	// Environment variables
	enableDebug = envutil.GetBool(EnvDebug, false)
	enableWatch = envutil.GetBool(EnvWatch, true)
	debugf("Config: Debug=%v, Watch=%v", enableDebug, enableWatch)

	// port value
	if portInt > 0 {
		port = fmt.Sprintf("%d", portInt)
	} else {
		port = envutil.Getenv(EnvPort, DefaultPort)
	}

	// Watch directory. multi use comma split
	if dirstr := envutil.Getenv(EnvWatchDir, ""); dirstr != "" {
		debugf("Config: Watch directory=%s", dirstr)
		watchDirs = strings.Split(dirstr, ",")
	}

	// Watch skip directory. multi use comma split
	if skipstr := envutil.Getenv(EnvWatchSkipDir, ""); skipstr != "" {
		if strings.HasPrefix(skipstr, "override") {
			watchSkipDirs = strings.Split(skipstr[10:], ",")
		} else {
			watchSkipDirs = append(defaultSkipDirs, strings.Split(skipstr, ",")...)
		}
		debugf("Config: Watch skip directory=%s", strings.Join(watchSkipDirs, ","))
	}
}

func watchDirectory(dir string) {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	// Walk and add all subdirectories
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// Skip directories start with dot or in watchSkipDirs
			if shouldSkipDir(name) {
				debugf("Skip watch directory: %s", name)
				return filepath.SkipDir
			}
			// watchDirs is not empty, only watch directories in watchDirs
			if len(watchDirs) > 0 && !slices.Contains(watchDirs, name) {
				debugf("Skip watch directory: %s", name)
				return filepath.SkipDir
			}

			debugf("Watch directory: %s", name)
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		clog.Error("Error walking directory:", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				if strings.HasSuffix(event.Name, ".md") {
					relPath, _ := filepath.Rel(dir, event.Name)
					clog.Warnf("Modified file: %s", relPath)
					broadcast("reload")
				}
			}
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					watcher.Add(event.Name)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			clog.Errorf("WATCH: Watcher error: %v", err)
		}
	}
}

func broadcast(msg string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		select {
		case client <- msg:
		default:
			// Client blocked, ignore
		}
	}
}
