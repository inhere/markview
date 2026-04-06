package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
