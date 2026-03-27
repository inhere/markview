package main

import (
	"bytes"
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
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
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
	port         string
	clients      = make(map[chan string]bool)
	clientsMu    sync.Mutex
	watcher      *fsnotify.Watcher
)

const (
	DefaultPort = "6100"
	DefaultEntry = "README.md"
)

type PageData struct {
	Title   string
	Content template.HTML
}

func main() {
	// 1. Configuration
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		binName := filepath.Base(os.Args[0])
		fmt.Printf("MarkView - Markdown Live Preview Server\n")
		fmt.Printf("  (Version: %s, Git Hash: %s, Build Time: %s)\n\n", Version, GitHash, BuildTime)
		fmt.Printf("Usage:\n")
		fmt.Printf("  %s [directory] [default-entry]\n\n", binName)
		fmt.Printf("Arguments:\n")
		fmt.Printf("  directory      Directory to watch (default: current dir)\n")
		fmt.Printf("  default-entry  Default markdown file to open (default: README.md)\n\n")
		fmt.Printf("Environment:\n")
		fmt.Printf("  SERVER_PORT    HTTP port to listen on (default: %s)\n", DefaultPort)
		return
	}

	// - Prepare arguments
	prepareArgs(args)

	fmt.Printf("Serving directory: %s\n", targetDir)
	fmt.Printf("Default entry file: %s\n", defaultEntry)
	fmt.Printf("🚀 Server running at http://localhost:%s\n", port)

	// 2. Watcher
	go watchDirectory(targetDir)

	// 3. HTTP Server
	distFS, err := fs.Sub(content, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(distFS))))

	http.HandleFunc("/sse", handleSSE)
	http.HandleFunc("/", handleRequest)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func prepareArgs(args []string) {

	cwd, _ := os.Getwd()
	targetDir = cwd
	if len(args) > 0 {
		absPath, err := filepath.Abs(args[0])
		if err == nil {
			targetDir = absPath
		}
	}

	defaultEntry = DefaultEntry
	if len(args) > 1 {
		defaultEntry = args[1]
	}

	port = os.Getenv("SERVER_PORT")
	if port == "" {
		port = DefaultPort
	}

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	var filePath string

	if urlPath == "/" {
		filePath = filepath.Join(targetDir, defaultEntry)
	} else {
		// Remove leading slash
		cleanPath := strings.TrimPrefix(urlPath, "/")
		filePath = filepath.Join(targetDir, cleanPath)
	}

	// Security check
	rel, err := filepath.Rel(targetDir, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "Access Denied", 403)
		return
	}

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.Error(w, "File Not Found: "+filePath, 404)
		return
	}

	if info.IsDir() {
		// Serve index.md if exists in directory
		indexPath := filepath.Join(filePath, "index.md")
		if _, err := os.Stat(indexPath); err == nil {
			filePath = indexPath
		} else {
			// List directory? For now just 404
			http.Error(w, "Directory listing not supported", 404)
			return
		}
	}

	if strings.HasSuffix(strings.ToLower(filePath), ".md") {
		renderMarkdown(w, filePath)
		return
	}

	// Serve static file
	http.ServeFile(w, r, filePath)
}

func renderMarkdown(w http.ResponseWriter, filePath string) {
	mdData, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "Failed to read file", 500)
		return
	}

	// Configure goldmark
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(), // Allow raw HTML
		),
	)

	var buf bytes.Buffer
	if err = md.Convert(mdData, &buf); err != nil {
		http.Error(w, "Failed to render markdown", 500)
		return
	}

	// Read template
	tmplData, err := content.ReadFile("frontend/template.html")
	if err != nil {
		http.Error(w, "Template not found", 500)
		return
	}

	// Use html/template properly
	t := template.Must(template.New("index").Parse(string(tmplData)))

	fileName := filepath.Base(filePath)
	data := PageData{
		Title:   fileName,
		Content: template.HTML(buf.String()),
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan string)

	clientsMu.Lock()
	clients[clientChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, clientChan)
		clientsMu.Unlock()
		close(clientChan)
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-time.After(15 * time.Second):
			// Keep alive
			fmt.Fprintf(w, ": keepalive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
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
			// Skip node_modules and .git
			if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		log.Println("Error walking directory:", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				if strings.HasSuffix(event.Name, ".md") {
					log.Println("Modified file:", event.Name)
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
			log.Println("Watcher error:", err)
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
