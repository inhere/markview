package bootstrap

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/projects"
)

func TestStaticHandlerSetsRevalidateCacheHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	rec := httptest.NewRecorder()

	newStaticHandler(testContentFS()).ServeHTTP(rec, req)

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

	cmd := newCommand(testOptions())
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
			cmd := newCommand(testOptions())
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
			cmd := newCommand(testOptions())
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

func TestPrepareLoadsDotenvFromTargetDir(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".env"), []byte("MKVIEW_PORT=6222\nMKVIEW_WATCH=false\n"), 0644))
	t.Setenv(config.EnvPort, "")
	t.Setenv(config.EnvWatch, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{}

	err := prepare([]string{targetDir}, testContentFS())

	assert.NoErr(t, err)
	assert.Eq(t, 6222, config.Cfg.PortInt)
	assert.Eq(t, config.PortSourceEnv, config.Cfg.PortSource)
	assert.False(t, config.Cfg.EnableWatch)
}

func TestPrepareProjectConfigPortSkipsProjectRegistryMode(t *testing.T) {
	targetDir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Test"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(targetDir, ".markview.json"), []byte(`{"server":{"port":6223}}`), 0644))
	t.Setenv(config.EnvPort, "")

	origCfg := config.Cfg
	t.Cleanup(func() { config.Cfg = origCfg })
	config.Cfg = config.Config{}

	err := prepare([]string{targetDir}, testContentFS())

	assert.NoErr(t, err)
	assert.Eq(t, 6223, config.Cfg.PortInt)
	assert.Eq(t, config.PortSourceConfig, config.Cfg.PortSource)
	assert.False(t, shouldUseProjectPortRegistry())
}

func TestListenNextAvailable(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoErr(t, err)
	defer occupied.Close()
	occupiedPort := occupied.Addr().(*net.TCPAddr).Port

	listener, actualPort, err := listenNextAvailable("127.0.0.1", occupiedPort, 3, nil)
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

func TestListenProjectPortFromRegistrySkipsPortsSavedByOtherProjects(t *testing.T) {
	ports, closePorts := reserveConsecutivePorts(t, 2)
	closePorts()
	targetDir := t.TempDir()
	otherDir := t.TempDir()

	registry := projects.Registry{}
	assert.NoErr(t, projects.Upsert(registry, targetDir, ports[0], nowForTest()))
	assert.NoErr(t, projects.Upsert(registry, otherDir, ports[0], nowForTest()))

	listener, actualPort, err := listenProjectPortFromRegistry("127.0.0.1", targetDir, registry, true)
	assert.NoErr(t, err)
	defer listener.Close()

	assert.Eq(t, ports[1], actualPort)
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

func reserveConsecutivePorts(t *testing.T, count int) ([]int, func()) {
	t.Helper()

	for start := 20000; start < 65000-count; start++ {
		listeners := make([]net.Listener, 0, count)
		ok := true
		for port := start; port < start+count; port++ {
			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				ok = false
				break
			}
			listeners = append(listeners, listener)
		}
		if !ok {
			for _, listener := range listeners {
				assert.NoErr(t, listener.Close())
			}
			continue
		}

		ports := make([]int, 0, count)
		for _, listener := range listeners {
			ports = append(ports, listener.Addr().(*net.TCPAddr).Port)
		}
		return ports, func() {
			for _, listener := range listeners {
				assert.NoErr(t, listener.Close())
			}
		}
	}

	t.Fatal("failed to reserve consecutive ports for test")
	return nil, nil
}

func nowForTest() time.Time {
	return time.Date(2026, 5, 14, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
}

func testOptions() options {
	return options{
		Content:   testContentFS(),
		Version:   "test",
		GitHash:   "test",
		BuildTime: "test",
	}
}

func testContentFS() fstest.MapFS {
	return fstest.MapFS{
		"web/dist/app.css": {Data: []byte("body{}")},
	}
}
