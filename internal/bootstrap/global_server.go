package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/projects"
)

type globalServer struct {
	manager      *ProjectManager
	registryPath string
	homeTemplate *template.Template
}

type globalProjectView struct {
	Name      string
	Path      string
	URL       string
	Added     string
	Available bool
}

type globalHomeData struct {
	Projects []globalProjectView
}

func newGlobalMux(manager *ProjectManager, content fs.FS, registryPath string) (http.Handler, error) {
	templateData, err := fs.ReadFile(content, "web/template-projects.html")
	if err != nil {
		return nil, err
	}
	homeTemplate, err := template.New("projects").Parse(string(templateData))
	if err != nil {
		return nil, err
	}
	server := &globalServer{manager: manager, registryPath: registryPath, homeTemplate: homeTemplate}
	mux := http.NewServeMux()
	mux.Handle("/static/", newStaticHandler(content))
	mux.Handle("/", server)
	return mux, nil
}

func (server *globalServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		server.renderHome(w)
		return
	}
	server.serveProject(w, r)
}

func (server *globalServer) renderHome(w http.ResponseWriter) {
	registry, index, err := server.loadIndex()
	if err != nil {
		http.Error(w, "Failed to load project registry", http.StatusInternalServerError)
		return
	}
	views := make([]globalProjectView, 0, len(registry))
	for _, entry := range projects.List(registry) {
		id, err := projects.StableID(entry.Path)
		if err != nil {
			http.Error(w, "Failed to index projects", http.StatusInternalServerError)
			return
		}
		project := index[id]
		name := entry.Record.Name
		if name == "" {
			name = filepath.Base(entry.Path)
		}
		views = append(views, globalProjectView{
			Name:      name,
			Path:      simplifyProjectPath(project.Path),
			URL:       "/p/" + id + "/",
			Added:     entry.Record.Added,
			Available: project.Exists,
		})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := server.homeTemplate.Execute(w, globalHomeData{Projects: views}); err != nil {
		http.Error(w, "Failed to render project registry", http.StatusInternalServerError)
	}
}

func (server *globalServer) serveProject(w http.ResponseWriter, r *http.Request) {
	escapedPath := strings.ToLower(r.URL.EscapedPath())
	if strings.Contains(escapedPath, "%2f") || strings.Contains(escapedPath, "%5c") || strings.Contains(escapedPath, "%00") {
		http.Error(w, "Invalid encoded path separator", http.StatusBadRequest)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "p" || !validProjectID(parts[1]) {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 {
		target := r.URL.Path + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
		return
	}
	_, index, err := server.loadIndex()
	if err != nil {
		writeGlobalProjectError(w, r, http.StatusInternalServerError, "registry_error", "Project registry is unavailable")
		return
	}
	project, ok := index[parts[1]]
	if !ok || !project.Exists {
		writeGlobalProjectError(w, r, http.StatusNotFound, "project_not_found", "Project is not available")
		return
	}
	runtime, err := server.manager.Runtime(r.Context(), project)
	if err != nil {
		writeGlobalProjectError(w, r, http.StatusInternalServerError, "project_init_failed", "Project failed to initialize")
		return
	}
	request := r.Clone(r.Context())
	requestURL := *r.URL
	requestURL.Path = "/" + strings.Join(parts[2:], "/")
	requestURL.RawPath = ""
	request.URL = &requestURL
	runtime.Server.ServeHTTP(w, request)
}

func (server *globalServer) loadIndex() (projects.Registry, projects.ProjectIndex, error) {
	registry, err := projects.Load(server.registryPath)
	if err != nil {
		return nil, nil, err
	}
	index, err := projects.BuildIndex(registry)
	return registry, index, err
}

func validProjectID(id string) bool {
	if len(id) != 12 {
		return false
	}
	for _, char := range id {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func writeGlobalProjectError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	if strings.HasPrefix(r.URL.Path, "/p/") && strings.Contains(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
		return
	}
	http.Error(w, message, status)
}

func simplifyProjectPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if rel, err := filepath.Rel(home, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.Join("~", rel)
		}
	}
	return path
}

func globalListenConfig() (config.Config, error) {
	globalCfg, _, err := config.LoadGlobalFileConfig()
	if err != nil {
		return config.Config{}, err
	}
	envCfg, err := config.RuntimeEnvConfig(nil)
	if err != nil {
		return config.Config{}, err
	}
	cliPort := (*int)(nil)
	if cliPortFlagVisited {
		cliPort = &config.Cfg.PortInt
	}
	merged, err := config.MergeRuntimeConfig(config.MergeInput{
		Global: globalCfg,
		Env:    config.EnvConfig{Port: envCfg.Port},
		CLI:    config.CLIConfig{Port: cliPort},
	})
	if err != nil {
		return config.Config{}, err
	}
	if merged.PortInt == 0 {
		merged.PortInt, err = strconv.Atoi(config.DefaultPort)
		if err != nil {
			return config.Config{}, fmt.Errorf("default port: %w", err)
		}
	}
	merged.Private = true
	if cliPrivateFlagVisited && !config.Cfg.Private {
		merged.Private = false
	}
	merged.NoBrowser = config.Cfg.NoBrowser
	return merged, nil
}

func validateGlobalMode(args []string) error {
	if len(args) > 0 {
		return errors.New("--global cannot be used with a directory or entry")
	}
	if selectedProject != "" {
		return errors.New("--global cannot be used with --project")
	}
	if projectsAction != "" {
		return errors.New("--global cannot be used with --projects")
	}
	return nil
}

func runGlobal(content fs.FS) error {
	listenerCfg, err := globalListenConfig()
	if err != nil {
		return err
	}
	registryPath, err := projectRegistryPath()
	if err != nil {
		return err
	}
	manager := NewProjectManager(content)
	handler, err := newGlobalMux(manager, content, registryPath)
	if err != nil {
		_ = manager.Close()
		return err
	}
	host := "0.0.0.0"
	if listenerCfg.Private {
		host = "127.0.0.1"
	}
	listener, actualPort, err := listenNextAvailable(host, listenerCfg.PortInt, 100, nil)
	if err != nil {
		_ = manager.Close()
		return err
	}
	listenerCfg.SetPort(actualPort)
	if !listenerCfg.Private {
		fmt.Println("WARNING: global server is public; every registered project may be accessible")
	}
	beforeServerRun(actualPort, listenerCfg.Private)

	server := &http.Server{
		Handler:     handler,
		ReadTimeout: 5 * time.Second,
		IdleTimeout: 120 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	serveErr := server.Serve(listener)
	stop()
	closeErr := manager.Close()
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, closeErr)
}
