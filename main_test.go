package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/projects"
)

func TestStaticHandlerSetsRevalidateCacheHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	rec := httptest.NewRecorder()

	newStaticHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=0, must-revalidate" {
		t.Fatalf("expected revalidate cache header, got %q", got)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/css") {
		t.Fatalf("expected css content-type, got %q", contentType)
	}
}

func TestBeforeServerRunOpensBrowserByDefault(t *testing.T) {
	origNoBrowser := config.Cfg.NoBrowser
	origOpenBrowser := openBrowser
	t.Cleanup(func() {
		config.Cfg.NoBrowser = origNoBrowser
		openBrowser = origOpenBrowser
	})

	var openedURL string
	openBrowser = func(url string) error {
		openedURL = url
		return nil
	}
	config.Cfg.NoBrowser = false

	beforeServerRun(6123, false)

	assert.Eq(t, "http://127.0.0.1:6123", openedURL)
}

func TestBeforeServerRunSkipsBrowserWhenDisabled(t *testing.T) {
	origNoBrowser := config.Cfg.NoBrowser
	origOpenBrowser := openBrowser
	t.Cleanup(func() {
		config.Cfg.NoBrowser = origNoBrowser
		openBrowser = origOpenBrowser
	})

	var opened bool
	openBrowser = func(string) error {
		opened = true
		return nil
	}
	config.Cfg.NoBrowser = true

	beforeServerRun(6123, false)

	assert.False(t, opened)
}

func TestMarkPortFlagVisited(t *testing.T) {
	origPortSource := config.Cfg.PortSource
	t.Cleanup(func() {
		config.Cfg.PortSource = origPortSource
	})

	cmd := newCommand()
	cmd.Func = nil
	err := cmd.Parse([]string{"-p", "-1"})
	assert.NoErr(t, err)

	markPortFlagVisited(cmd)

	assert.Eq(t, config.PortSourceCLI, config.Cfg.PortSource)
}

func TestProjectFlagParse(t *testing.T) {
	origSelectedProject := selectedProject
	t.Cleanup(func() {
		selectedProject = origSelectedProject
	})

	tests := []struct {
		name string
		args []string
	}{
		{name: "short project flag", args: []string{"-P", "markview"}},
		{name: "long project flag", args: []string{"--project", "markview"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedProject = ""
			cmd := newCommand()
			cmd.Func = nil

			err := cmd.Parse(tt.args)

			assert.NoErr(t, err)
			assert.Eq(t, "markview", selectedProject)
		})
	}
}

func TestResolveSelectedProjectTarget(t *testing.T) {
	t.Run("resolves project by name", func(t *testing.T) {
		projectDir := t.TempDir()
		withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(_ string) {
			targetDir, err := resolveSelectedProjectTarget("markview")

			assert.NoErr(t, err)
			assert.Eq(t, projectDir, targetDir)
		})
	})

	t.Run("resolves project by full path", func(t *testing.T) {
		projectDir := t.TempDir()
		withTempProjectRegistry(t, registryForTest(t, projectDir, "markview", 6100), func(_ string) {
			targetDir, err := resolveSelectedProjectTarget(projectDir)

			assert.NoErr(t, err)
			assert.Eq(t, projectDir, targetDir)
		})
	})

	t.Run("returns error for unknown project", func(t *testing.T) {
		withTempProjectRegistry(t, projects.Registry{}, func(_ string) {
			_, err := resolveSelectedProjectTarget("missing")

			assert.Err(t, err)
		})
	})

	t.Run("returns error for ambiguous project", func(t *testing.T) {
		registry := projects.Registry{
			"/projects/a/docs": {Name: "docs", Port: 6100, Added: "2026-05-14T15:00:00+08:00"},
			"/projects/b/docs": {Name: "docs", Port: 6101, Added: "2026-05-14T15:00:00+08:00"},
		}
		withTempProjectRegistry(t, registry, func(_ string) {
			_, err := resolveSelectedProjectTarget("docs")

			assert.Err(t, err)
		})
	})
}

func TestBuildPrepareArgsForSelectedProject(t *testing.T) {
	t.Run("uses selected project path with default entry", func(t *testing.T) {
		projectDir := t.TempDir()
		args, err := buildPrepareArgsForSelectedProject(projectDir, nil)

		assert.NoErr(t, err)
		assert.Eq(t, []string{projectDir}, args)
	})

	t.Run("uses one positional argument as entry file", func(t *testing.T) {
		projectDir := t.TempDir()
		args, err := buildPrepareArgsForSelectedProject(projectDir, []string{"guide.md"})

		assert.NoErr(t, err)
		assert.Eq(t, []string{projectDir, "guide.md"}, args)
	})

	t.Run("rejects multiple positional arguments", func(t *testing.T) {
		projectDir := t.TempDir()
		_, err := buildPrepareArgsForSelectedProject(projectDir, []string{"docs", "guide.md"})

		assert.Err(t, err)
	})
}

func TestShouldUseProjectPortRegistry(t *testing.T) {
	origPortSource := config.Cfg.PortSource
	origPortInt := config.Cfg.PortInt
	t.Cleanup(func() {
		config.Cfg.PortSource = origPortSource
		config.Cfg.PortInt = origPortInt
	})

	tests := []struct {
		name      string
		source    config.PortSource
		port      int
		shouldUse bool
	}{
		{name: "unset port uses registry", source: config.PortSourceUnset, port: 6100, shouldUse: true},
		{name: "CLI random uses registry", source: config.PortSourceCLI, port: -1, shouldUse: true},
		{name: "CLI fixed skips registry", source: config.PortSourceCLI, port: 8080, shouldUse: false},
		{name: "ENV fixed skips registry", source: config.PortSourceEnv, port: 8080, shouldUse: false},
		{name: "ENV random skips registry", source: config.PortSourceEnv, port: -1, shouldUse: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Cfg.PortSource = tt.source
			config.Cfg.PortInt = tt.port

			assert.Eq(t, tt.shouldUse, shouldUseProjectPortRegistry())
		})
	}
}

func TestProjectFlagKeepsPortRegistryRules(t *testing.T) {
	origSelectedProject := selectedProject
	origPortSource := config.Cfg.PortSource
	origPortInt := config.Cfg.PortInt
	t.Cleanup(func() {
		selectedProject = origSelectedProject
		config.Cfg.PortSource = origPortSource
		config.Cfg.PortInt = origPortInt
	})

	tests := []struct {
		name      string
		args      []string
		envPort   string
		shouldUse bool
	}{
		{name: "project without explicit port uses registry", args: []string{"-P", "docs"}, shouldUse: true},
		{name: "project with CLI random uses registry", args: []string{"-P", "docs", "-p", "-1"}, shouldUse: true},
		{name: "project with CLI fixed port skips registry", args: []string{"-P", "docs", "-p", "8080"}, shouldUse: false},
		{name: "project with ENV fixed port skips registry", args: []string{"-P", "docs"}, envPort: "8080", shouldUse: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectedProject = ""
			config.Cfg.PortSource = config.PortSourceUnset
			config.Cfg.PortInt = 0
			t.Setenv(config.EnvPort, tt.envPort)
			cmd := newCommand()
			cmd.Func = nil

			err := cmd.Parse(tt.args)
			assert.NoErr(t, err)
			markPortFlagVisited(cmd)
			if tt.envPort != "" && config.Cfg.PortSource != config.PortSourceCLI {
				config.Cfg.PortSource = config.PortSourceEnv
			}

			assert.Eq(t, tt.shouldUse, shouldUseProjectPortRegistry())
		})
	}
}

func TestListenNextAvailable(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer occupied.Close()
	occupiedPort := occupied.Addr().(*net.TCPAddr).Port

	listener, actualPort, err := listenNextAvailable("127.0.0.1", occupiedPort, 3)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, occupiedPort+1, actualPort)
}

func TestListenProjectPortFromRegistryUsesSavedPort(t *testing.T) {
	targetDir := t.TempDir()
	seed, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	savedPort := seed.Addr().(*net.TCPAddr).Port
	seed.Close()

	registry := projects.Registry{}
	err = projects.Upsert(registry, targetDir, savedPort, nowForTest())
	assert.NoErr(t, err)

	listener, actualPort, err := listenProjectPortFromRegistry("127.0.0.1", targetDir, registry, true)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, savedPort, actualPort)
}

func TestListenProjectPortFromRegistryFallsThroughFromDefaultPort(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:6100")
	if err != nil {
		t.Skipf("port 6100 is already occupied by another process: %v", err)
	}
	defer occupied.Close()

	listener, actualPort, err := listenProjectPortFromRegistry("127.0.0.1", t.TempDir(), projects.Registry{}, true)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, 6101, actualPort)
}

func nowForTest() time.Time {
	return time.Date(2026, 5, 14, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
}
