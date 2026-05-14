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
