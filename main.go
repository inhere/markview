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
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/goutil/cflag"
	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/x/clog"
)

//go:embed frontend/template.html frontend/dist
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
)

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
)

type PageData struct {
	Title               string
	Content             template.HTML
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

	cmd.IntVar(&portInt, "port", 0, "port for the live server;;p")
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
	go watchDirectory(targetDir)

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
	distFS, err := fs.Sub(content, "frontend/dist")
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

	// port value
	if portInt > 0 {
		port = fmt.Sprintf("%d", portInt)
	} else {
		port = envutil.Getenv(EnvPort, DefaultPort)
	}
}

var skipDirNames = []string{
	"node_modules",
	"dist",
	"tmp",
	"temp",
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
			// Skip directories start with dot or in skipDirNames
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
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
					clog.Warnf("WATCH: Modified file: %s", event.Name)
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

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
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
