package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/markview/internal/config"
)

func TestProjectServersKeepRootsIndependent(t *testing.T) {
	serverA := newRawTestProjectServer(t, "# Project A")
	serverB := newRawTestProjectServer(t, "# Project B")

	for _, tc := range []struct {
		name   string
		server http.Handler
		want   string
	}{
		{name: "A", server: serverA, want: "# Project A"},
		{name: "B", server: serverB, want: "# Project B"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tc.server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/?q=raw", nil))

			assert.Eq(t, http.StatusOK, recorder.Code)
			assert.Eq(t, tc.want, recorder.Body.String())
		})
	}
}

func TestProjectServersKeepSearchIndependent(t *testing.T) {
	serverA := newRawTestProjectServer(t, "# unique-alpha")
	serverB := newRawTestProjectServer(t, "# unique-beta")

	recorderA := httptest.NewRecorder()
	serverA.ServeHTTP(recorderA, httptest.NewRequest(http.MethodGet, "/api/search?q=unique-alpha", nil))
	recorderB := httptest.NewRecorder()
	serverB.ServeHTTP(recorderB, httptest.NewRequest(http.MethodGet, "/api/search?q=unique-alpha", nil))

	assert.Eq(t, http.StatusOK, recorderA.Code)
	assert.True(t, strings.Contains(recorderA.Body.String(), "README.md"))
	assert.Eq(t, http.StatusOK, recorderB.Code)
	assert.False(t, strings.Contains(recorderB.Body.String(), "README.md"))
}

func TestProjectServerRendersFullAndMainHTML(t *testing.T) {
	server := newRawTestProjectServer(t, "# Project A")
	for _, target := range []string{"/", "/?q=main"} {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

			assert.Eq(t, http.StatusOK, recorder.Code)
			assert.True(t, strings.Contains(recorder.Body.String(), "Project A"))
			assert.Eq(t, "no-store", recorder.Header().Get("Cache-Control"))
		})
	}
}

func TestProjectServerReturnsOwnFileTree(t *testing.T) {
	server := newRawTestProjectServer(t, "# Project A")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/file-tree", nil))

	assert.Eq(t, http.StatusOK, recorder.Code)
	assert.True(t, strings.Contains(recorder.Body.String(), "README.md"))
	assert.Eq(t, "no-store", recorder.Header().Get("Cache-Control"))
}

func TestProjectServersKeepAppConfigIndependent(t *testing.T) {
	serverA := newRawTestProjectServer(t, "# A")
	serverB := newRawTestProjectServer(t, "# B")
	serverA.Config.UILayout = config.UILayoutCompact
	serverB.Config.UILayout = config.UILayoutTOCRight

	recorderA := httptest.NewRecorder()
	serverA.ServeHTTP(recorderA, httptest.NewRequest(http.MethodGet, "/", nil))
	recorderB := httptest.NewRecorder()
	serverB.ServeHTTP(recorderB, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, strings.Contains(recorderA.Body.String(), config.UILayoutCompact))
	assert.True(t, strings.Contains(recorderB.Body.String(), config.UILayoutTOCRight))
}

func TestProjectServerFileTreeSkipsOutsideJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junction test is Windows-only")
	}
	dir := t.TempDir()
	outside := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Safe"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0644))
	junction := filepath.Join(dir, "outside")
	output, err := exec.Command("cmd", "/c", "mklink", "/J", junction, outside).CombinedOutput()
	if err != nil {
		t.Skipf("junction unavailable: %v: %s", err, output)
	}
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)
	server := NewProjectServer(config.Config{TargetDir: dir, EntryFile: "README.md"}, root, NewEventHub(), fstest.MapFS{})
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/file-tree", nil))

	assert.Eq(t, http.StatusOK, recorder.Code)
	assert.False(t, strings.Contains(recorder.Body.String(), "secret.md"))
}

func TestProjectServerRootEntryDirectoryUsesItsIndex(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.Mkdir(filepath.Join(dir, "docs"), 0755))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "docs", "index.md"), []byte("# Docs index"), 0644))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)
	server := NewProjectServer(
		config.Config{TargetDir: dir, EntryFile: "docs"},
		root,
		NewEventHub(),
		fstest.MapFS{},
	)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/?q=raw", nil))

	assert.Eq(t, http.StatusOK, recorder.Code)
	assert.Eq(t, "# Docs index", recorder.Body.String())
}

func newRawTestProjectServer(t *testing.T, body string) *ProjectServer {
	t.Helper()
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(body), 0644))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)
	cfg := config.Config{
		TargetDir:     dir,
		EntryFile:     "README.md",
		WatchSkipDirs: append([]string(nil), config.DefaultSkipDirs...),
	}
	content := fstest.MapFS{
		"web/template-main.html": {Data: []byte(`{{.Content}}`)},
		"web/template.html":      {Data: []byte(`{{.AppConfigJSON}}{{.MainContent}}`)},
	}
	return NewProjectServer(cfg, root, NewEventHub(), content)
}
