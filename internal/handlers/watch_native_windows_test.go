//go:build windows

package handlers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/markview/internal/config"
)

// TestWindowsWatcherAllowsRenamingWatchedSubdirectory 回归测试：
// 单句柄递归监听不应再锁住子目录，且重命名后仍能收到变更。
func TestWindowsWatcherAllowsRenamingWatchedSubdirectory(t *testing.T) {
	watcher, hub, dir := newRunningTestWatcher(t)
	defer watcher.Close()
	client, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	nested := filepath.Join(dir, "level1", "level2")
	assert.NoErr(t, os.MkdirAll(nested, 0755))
	time.Sleep(200 * time.Millisecond)

	renamedDir := filepath.Join(dir, "level1-renamed")
	assert.NoErr(t, os.Rename(filepath.Join(dir, "level1"), renamedDir))

	assert.NoErr(t, os.WriteFile(filepath.Join(renamedDir, "level2", "after.md"), []byte("# after"), 0644))
	select {
	case message := <-client:
		assert.True(t, strings.Contains(message, "level1-renamed/level2/after.md"))
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event after rename")
	}
}

func TestWindowsWatcherReceivesDeepMarkdownChanges(t *testing.T) {
	_, hub, dir := newRunningTestWatcher(t)
	client, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	deep := filepath.Join(dir, "a", "b", "c")
	assert.NoErr(t, os.MkdirAll(deep, 0755))
	assert.NoErr(t, os.WriteFile(filepath.Join(deep, "note.md"), []byte("# note"), 0644))

	select {
	case message := <-client:
		assert.True(t, strings.Contains(message, "a/b/c/note.md"))
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for deep event")
	}
}

func TestWindowsWatcherSkipsConfiguredDirectories(t *testing.T) {
	dir := t.TempDir()
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)
	hub := NewEventHub()
	watcher, err := NewWatcher(root, config.Config{TargetDir: dir}, hub)
	assert.NoErr(t, err)
	watcher.debounce = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer watcher.Close()
	go func() { _ = watcher.Run(ctx) }()

	client, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	skipped := filepath.Join(dir, "node_modules")
	assert.NoErr(t, os.MkdirAll(skipped, 0755))
	assert.NoErr(t, os.WriteFile(filepath.Join(skipped, "dep.md"), []byte("dep"), 0644))

	select {
	case message := <-client:
		t.Fatalf("skipped directory produced event: %s", message)
	case <-time.After(500 * time.Millisecond):
	}
}
