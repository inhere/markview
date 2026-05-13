package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/inhere/markview/internal/config"
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
