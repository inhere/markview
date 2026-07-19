package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestLoadProjectRuntimeConfigGlobalModeKeepsContentAndIgnoresListenerSettings(t *testing.T) {
	dir := t.TempDir()
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("MKVIEW_PORT=6222\nMKVIEW_ENTRY=guide.md\nMKVIEW_WATCH=false\n"), 0644))
	assert.NoErr(t, os.WriteFile(filepath.Join(dir, "markview.json"), []byte(`{
  "server": {"private": true},
  "ui": {"layout": "toc-right"}
}`), 0644))
	t.Setenv(EnvPort, "")
	t.Setenv(EnvEntry, "")
	t.Setenv(EnvWatch, "")
	original := Cfg
	t.Cleanup(func() { Cfg = original })
	Cfg = Config{TargetDir: "unchanged"}

	got, err := LoadProjectRuntimeConfig(dir, ProjectLoadOptions{GlobalMode: true})

	assert.NoErr(t, err)
	assert.Eq(t, 0, got.PortInt)
	assert.Eq(t, PortSourceUnset, got.PortSource)
	assert.False(t, got.Private)
	assert.Eq(t, "guide.md", got.EntryFile)
	assert.False(t, got.EnableWatch)
	assert.Eq(t, UILayoutTOCRight, got.UILayout)
	assert.Eq(t, "unchanged", Cfg.TargetDir)
	assert.Eq(t, "", os.Getenv(EnvEntry))
}

func TestLoadProjectRuntimeConfigConcurrentProjectsStayIndependent(t *testing.T) {
	t.Setenv(EnvEntry, "")
	type result struct {
		entry string
		err   error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, entry := range []string{"a.md", "b.md"} {
		dir := t.TempDir()
		assert.NoErr(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("MKVIEW_ENTRY="+entry+"\n"), 0644))
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, err := LoadProjectRuntimeConfig(dir, ProjectLoadOptions{GlobalMode: true})
			results <- result{entry: cfg.EntryFile, err: err}
		}()
	}
	wg.Wait()
	close(results)

	seen := map[string]bool{}
	for item := range results {
		assert.NoErr(t, item.err)
		seen[item.entry] = true
	}
	assert.True(t, seen["a.md"])
	assert.True(t, seen["b.md"])
	assert.Eq(t, "", os.Getenv(EnvEntry))
}
