package bootstrap

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/projects"
)

func TestGlobalMuxServesRegistryProjectsAndRefreshesRegistry(t *testing.T) {
	projectDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Global project"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(projectDir, ".env"), []byte("MKVIEW_WATCH=false\n"), 0644))
	registryPath := filepath.Join(t.TempDir(), "projects.json")
	registry := projects.Registry{}
	assert.NoErr(t, projects.Upsert(registry, projectDir, 6100, time.Now()))
	assert.NoErr(t, projects.Save(registryPath, registry))
	projectID, err := projects.StableID(projectDir)
	assert.NoErr(t, err)
	content := globalTestContent()
	manager := NewProjectManager(content)
	t.Cleanup(func() { _ = manager.Close() })
	handler, err := newGlobalMux(manager, content, registryPath)
	assert.NoErr(t, err)

	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Eq(t, http.StatusOK, home.Code)
	assert.True(t, strings.Contains(home.Body.String(), filepath.Base(projectDir)))

	redirect := httptest.NewRecorder()
	handler.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/p/"+projectID+"?q=raw", nil))
	assert.Eq(t, http.StatusPermanentRedirect, redirect.Code)
	assert.Eq(t, "/p/"+projectID+"/?q=raw", redirect.Header().Get("Location"))

	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/p/"+projectID+"/?q=raw", nil))
	assert.Eq(t, http.StatusOK, page.Code)
	assert.Eq(t, "# Global project", page.Body.String())
	fullPage := httptest.NewRecorder()
	handler.ServeHTTP(fullPage, httptest.NewRequest(http.MethodGet, "/p/"+projectID+"/", nil))
	assert.Eq(t, http.StatusOK, fullPage.Code)
	assert.True(t, strings.Contains(fullPage.Body.String(), `"basePath":"/p/`+projectID+`"`))

	assert.NoErr(t, projects.Save(registryPath, projects.Registry{}))
	removed := httptest.NewRecorder()
	handler.ServeHTTP(removed, httptest.NewRequest(http.MethodGet, "/p/"+projectID+"/?q=raw", nil))
	assert.Eq(t, http.StatusNotFound, removed.Code)
}

func TestGlobalMuxRejectsEncodedSeparators(t *testing.T) {
	content := globalTestContent()
	manager := NewProjectManager(content)
	t.Cleanup(func() { _ = manager.Close() })
	registryPath := filepath.Join(t.TempDir(), "projects.json")
	assert.NoErr(t, projects.Save(registryPath, projects.Registry{}))
	handler, err := newGlobalMux(manager, content, registryPath)
	assert.NoErr(t, err)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/p/aaaaaaaaaaaa/docs%2Fsecret.md", nil))

	assert.Eq(t, http.StatusBadRequest, recorder.Code)
}

func TestGlobalListenConfigRequiresExplicitPrivateFalseToPublish(t *testing.T) {
	originalCfg := config.Cfg
	originalPortVisited := cliPortFlagVisited
	originalPrivateVisited := cliPrivateFlagVisited
	t.Cleanup(func() {
		config.Cfg = originalCfg
		cliPortFlagVisited = originalPortVisited
		cliPrivateFlagVisited = originalPrivateVisited
	})
	t.Setenv(config.EnvPort, "")
	config.Cfg = config.Config{}
	cliPortFlagVisited = false
	cliPrivateFlagVisited = false

	privateCfg, err := globalListenConfig()
	assert.NoErr(t, err)
	assert.True(t, privateCfg.Private)
	assert.Eq(t, 6100, privateCfg.PortInt)

	config.Cfg.Private = false
	cliPrivateFlagVisited = true
	publicCfg, err := globalListenConfig()
	assert.NoErr(t, err)
	assert.False(t, publicCfg.Private)
}

func TestValidateGlobalModeRejectsConflictingTargets(t *testing.T) {
	originalProject := selectedProject
	originalAction := projectsAction
	t.Cleanup(func() {
		selectedProject = originalProject
		projectsAction = originalAction
	})
	for _, tc := range []struct {
		name    string
		args    []string
		project string
		action  string
	}{
		{name: "positional directory", args: []string{"docs"}},
		{name: "selected project", project: "docs"},
		{name: "projects action", action: "list"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selectedProject, projectsAction = tc.project, tc.action
			assert.Err(t, validateGlobalMode(tc.args))
		})
	}
	selectedProject, projectsAction = "", ""
	assert.NoErr(t, validateGlobalMode(nil))
}

func TestGlobalFlagParse(t *testing.T) {
	original := globalMode
	t.Cleanup(func() { globalMode = original })
	globalMode = false
	cmd := newCommand(testOptions())
	cmd.Func = nil

	err := cmd.Parse([]string{"--global"})

	assert.NoErr(t, err)
	assert.True(t, globalMode)
}

func TestGlobalProjectsTemplateParses(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "web", "template-projects.html"))
	assert.NoErr(t, err)

	_, err = template.New("projects").Parse(string(data))

	assert.NoErr(t, err)
}

func globalTestContent() fstest.MapFS {
	return fstest.MapFS{
		"web/template-projects.html": {Data: []byte(`{{range .Projects}}{{.Name}} {{.URL}} {{.Available}}{{end}}`)},
		"web/template-main.html":     {Data: []byte(`{{.Content}}`)},
		"web/template.html":          {Data: []byte(`<script type="application/json">{{.AppConfigJSON}}</script>{{.MainContent}}`)},
		"web/dist/app.css":           {Data: []byte("body{}")},
	}
}
