package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFileTree(t *testing.T) {
	root := t.TempDir()

	files := []string{
		"README.md",
		"notes.txt",
		"guide/index.md",
		"guide/intro.md",
		"guide/images/logo.png",
		"guide/deep/topic.md",
		"guide/deep/index.md",
		"plain/child.md",
		"z-last.md",
	}

	for _, relativePath := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", relativePath, err)
		}
		if err := os.WriteFile(fullPath, []byte("# test"), 0o644); err != nil {
			t.Fatalf("write %s: %v", relativePath, err)
		}
	}

	tree, err := buildFileTree(root)
	if err != nil {
		t.Fatalf("buildFileTree returned error: %v", err)
	}

	if len(tree) != 4 {
		t.Fatalf("expected 4 root nodes, got %d", len(tree))
	}

	if tree[0].Kind != "directory" || tree[0].Name != "guide" {
		t.Fatalf("expected first node to be guide directory, got %+v", tree[0])
	}
	if !tree[0].Navigable {
		t.Fatalf("expected guide directory to be navigable")
	}
	if tree[0].Href != "/guide" {
		t.Fatalf("expected guide href /guide, got %s", tree[0].Href)
	}
	if tree[0].MatchPath != "guide/index.md" {
		t.Fatalf("expected guide matchPath guide/index.md, got %s", tree[0].MatchPath)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected guide to expose 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Kind != "directory" || tree[0].Children[0].Name != "deep" {
		t.Fatalf("expected guide child 0 to be deep directory, got %+v", tree[0].Children[0])
	}
	if tree[0].Children[1].Kind != "file" || tree[0].Children[1].Name != "intro.md" {
		t.Fatalf("expected guide child 1 to be intro.md, got %+v", tree[0].Children[1])
	}

	if tree[1].Kind != "directory" || tree[1].Name != "plain" {
		t.Fatalf("expected second node to be plain directory, got %+v", tree[1])
	}
	if tree[1].Navigable {
		t.Fatalf("expected plain directory to be non-navigable without index.md")
	}
	if len(tree[1].Children) != 1 || tree[1].Children[0].Name != "child.md" {
		t.Fatalf("expected plain to contain child.md, got %+v", tree[1].Children)
	}

	if tree[2].Kind != "file" || tree[2].Name != "README.md" {
		t.Fatalf("expected third node to be README.md, got %+v", tree[2])
	}
	if tree[3].Kind != "file" || tree[3].Name != "z-last.md" {
		t.Fatalf("expected fourth node to be z-last.md, got %+v", tree[3])
	}
}

func TestHandleRequestSetsNoStoreForMarkdownPages(t *testing.T) {
	root := t.TempDir()
	targetDir = root
	defaultEntry = "README.md"

	readmePath := filepath.Join(root, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Hello"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", got)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected html content-type, got %q", contentType)
	}
}

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
