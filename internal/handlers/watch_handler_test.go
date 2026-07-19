package handlers

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/markview/internal/config"
)

func TestWatchersPublishOnlyToTheirProjectHub(t *testing.T) {
	watcherA, hubA, dirA := newRunningTestWatcher(t)
	watcherB, hubB, _ := newRunningTestWatcher(t)
	clientA, cancelA := hubA.Subscribe()
	defer cancelA()
	clientB, cancelB := hubB.Subscribe()
	defer cancelB()

	assert.NoErr(t, os.WriteFile(filepath.Join(dirA, "changed.md"), []byte("# changed"), 0644))
	select {
	case message := <-clientA:
		assert.True(t, strings.Contains(message, "changed.md"))
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for project A event")
	}
	select {
	case message := <-clientB:
		t.Fatalf("event leaked to project B: %s", message)
	case <-time.After(100 * time.Millisecond):
	}
	assert.NoErr(t, watcherA.Close())
	assert.NoErr(t, watcherA.Close())
	assert.NoErr(t, watcherB.Close())
}

func TestWatcherRejectsRuntimeOutsideJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junction test is Windows-only")
	}
	watcher, hub, dir := newRunningTestWatcher(t)
	defer watcher.Close()
	client, cancel := hub.Subscribe()
	defer cancel()
	outside := t.TempDir()
	junction := filepath.Join(dir, "outside")
	output, err := exec.Command("cmd", "/c", "mklink", "/J", junction, outside).CombinedOutput()
	if err != nil {
		t.Skipf("junction unavailable: %v: %s", err, output)
	}
	time.Sleep(100 * time.Millisecond)
	assert.NoErr(t, os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0644))

	select {
	case message := <-client:
		t.Fatalf("outside junction produced event: %s", message)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestWatcherAddsRuntimeDirectoryInsideConfiguredWatchDir(t *testing.T) {
	dir := t.TempDir()
	docs := filepath.Join(dir, "docs")
	assert.NoErr(t, os.Mkdir(docs, 0755))
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)
	hub := NewEventHub()
	watcher, err := NewWatcher(root, config.Config{TargetDir: dir, WatchDirs: []string{"docs"}}, hub)
	assert.NoErr(t, err)
	watcher.debounce = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer watcher.Close()
	go func() { _ = watcher.Run(ctx) }()
	client, unsubscribe := hub.Subscribe()
	defer unsubscribe()

	nested := filepath.Join(docs, "new")
	assert.NoErr(t, os.Mkdir(nested, 0755))
	time.Sleep(100 * time.Millisecond)
	assert.NoErr(t, os.WriteFile(filepath.Join(nested, "added.md"), []byte("added"), 0644))

	select {
	case message := <-client:
		assert.True(t, strings.Contains(message, "docs/new/added.md"))
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for nested watch event")
	}
}

func newRunningTestWatcher(t *testing.T) (*Watcher, *EventHub, string) {
	t.Helper()
	dir := t.TempDir()
	root, err := NewProjectRoot(dir)
	assert.NoErr(t, err)
	hub := NewEventHub()
	watcher, err := NewWatcher(root, config.Config{TargetDir: dir}, hub)
	assert.NoErr(t, err)
	watcher.debounce = 20 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		watcher.Close()
		hub.Close()
	})
	go func() { _ = watcher.Run(ctx) }()
	return watcher, hub, dir
}
