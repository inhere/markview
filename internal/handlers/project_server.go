package handlers

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inhere/markview/internal/config"
)

type ProjectServer struct {
	Config  config.Config
	Root    ProjectRoot
	Events  *EventHub
	content fs.FS
	mux     *http.ServeMux
}

func NewProjectServer(cfg config.Config, root ProjectRoot, events *EventHub, content fs.FS) *ProjectServer {
	if events == nil {
		events = defaultEventHub
	}
	server := &ProjectServer{Config: cfg, Root: root, Events: events, content: content}
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", events.HandleSSE)
	mux.Handle("/api/search", http.TimeoutHandler(http.HandlerFunc(server.handleSearch), 10*time.Second, "request timeout"))
	mux.Handle("/api/file-tree", http.TimeoutHandler(http.HandlerFunc(server.handleFileTree), 10*time.Second, "request timeout"))
	mux.HandleFunc("/", server.handlePage)
	server.mux = mux
	return server
}

func (server *ProjectServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.mux.ServeHTTP(w, r)
}

func (server *ProjectServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	handleSearchForConfig(w, r, server.Config)
}

func (server *ProjectServer) handleFileTree(w http.ResponseWriter, r *http.Request) {
	handleFileTreeForConfig(w, r, server.Config)
}

func (server *ProjectServer) handlePage(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	originalPath := urlPath
	if urlPath == "/" {
		urlPath = "/" + strings.TrimPrefix(filepath.ToSlash(server.Config.EntryFile), "/")
	}
	filePath, err := server.Root.Resolve(urlPath)
	if err != nil {
		if os.IsNotExist(err) {
			if originalPath == "/" {
				renderDirectoryListingForProject(w, r, server.Root.RealPath, server.Config, server.readContent)
				return
			}
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Access Denied", http.StatusForbidden)
		return
	}
	info, err := os.Stat(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if info.IsDir() {
		indexURL := strings.TrimSuffix(urlPath, "/") + "/index.md"
		indexPath, indexErr := server.Root.Resolve(indexURL)
		if indexErr == nil {
			filePath = indexPath
		} else {
			renderDirectoryListingForProject(w, r, filePath, server.Config, server.readContent)
			return
		}
	}
	if !strings.HasSuffix(strings.ToLower(filePath), ".md") {
		http.ServeFile(w, r, filePath)
		return
	}
	switch r.URL.Query().Get("q") {
	case queryTypeMain:
		renderMainContentForProject(w, filePath, server.Config.TargetDir, server.readContent)
	case queryTypeRaw:
		renderRawMarkdown(w, filePath)
	default:
		renderMarkdownForProject(w, filePath, server.Config, server.readContent)
	}
}

func (server *ProjectServer) readContent(path string) ([]byte, error) {
	return fs.ReadFile(server.content, path)
}
