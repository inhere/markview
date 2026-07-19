package bootstrap

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gookit/goutil/x/assert"
	"github.com/inhere/markview/internal/handlers"
	"github.com/inhere/markview/internal/projects"
)

func TestProjectManagerInitializesProjectOnce(t *testing.T) {
	manager := NewProjectManager(fstest.MapFS{})
	var calls atomic.Int32
	manager.factory = func(context.Context, projects.IndexedProject) (*ProjectRuntime, error) {
		calls.Add(1)
		return &ProjectRuntime{Events: handlers.NewEventHub()}, nil
	}
	project := projects.IndexedProject{ID: "aaaaaaaaaaaa", Path: t.TempDir(), Exists: true}
	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.Runtime(context.Background(), project)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoErr(t, err)
	}
	assert.Eq(t, int32(1), calls.Load())
	assert.NoErr(t, manager.Close())
}

func TestProjectManagerRetriesInitializationFailure(t *testing.T) {
	manager := NewProjectManager(fstest.MapFS{})
	var calls atomic.Int32
	manager.factory = func(context.Context, projects.IndexedProject) (*ProjectRuntime, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("first failure")
		}
		return &ProjectRuntime{Events: handlers.NewEventHub()}, nil
	}
	project := projects.IndexedProject{ID: "aaaaaaaaaaaa", Path: t.TempDir(), Exists: true}

	_, firstErr := manager.Runtime(context.Background(), project)
	runtime, secondErr := manager.Runtime(context.Background(), project)

	assert.Err(t, firstErr)
	assert.NoErr(t, secondErr)
	assert.True(t, runtime != nil)
	assert.Eq(t, int32(2), calls.Load())
	assert.NoErr(t, manager.Close())
}

func TestProjectManagerInitializesDifferentProjectsConcurrently(t *testing.T) {
	manager := NewProjectManager(fstest.MapFS{})
	started := make(chan string, 2)
	release := make(chan struct{})
	manager.factory = func(_ context.Context, project projects.IndexedProject) (*ProjectRuntime, error) {
		started <- project.ID
		<-release
		return &ProjectRuntime{Events: handlers.NewEventHub()}, nil
	}
	projectsToLoad := []projects.IndexedProject{
		{ID: "aaaaaaaaaaaa", Path: t.TempDir(), Exists: true},
		{ID: "bbbbbbbbbbbb", Path: t.TempDir(), Exists: true},
	}
	errs := make(chan error, 2)
	for _, project := range projectsToLoad {
		go func() {
			_, err := manager.Runtime(context.Background(), project)
			errs <- err
		}()
	}
	seen := map[string]bool{<-started: true, <-started: true}
	close(release)

	assert.True(t, seen["aaaaaaaaaaaa"])
	assert.True(t, seen["bbbbbbbbbbbb"])
	assert.NoErr(t, <-errs)
	assert.NoErr(t, <-errs)
	assert.NoErr(t, manager.Close())
}

func TestProjectManagerCloseRejectsInflightRuntime(t *testing.T) {
	manager := NewProjectManager(fstest.MapFS{})
	started := make(chan struct{})
	release := make(chan struct{})
	created := make(chan *ProjectRuntime, 1)
	manager.factory = func(context.Context, projects.IndexedProject) (*ProjectRuntime, error) {
		close(started)
		<-release
		runtime := &ProjectRuntime{Events: handlers.NewEventHub()}
		created <- runtime
		return runtime, nil
	}
	project := projects.IndexedProject{ID: "aaaaaaaaaaaa", Path: t.TempDir(), Exists: true}
	runtimeErr := make(chan error, 1)
	go func() {
		_, err := manager.Runtime(context.Background(), project)
		runtimeErr <- err
	}()
	<-started
	closeErr := make(chan error, 1)
	go func() { closeErr <- manager.Close() }()
	for {
		manager.mu.Lock()
		closed := manager.closed
		manager.mu.Unlock()
		if closed {
			break
		}
		time.Sleep(time.Millisecond)
	}
	close(release)

	assert.NoErr(t, <-closeErr)
	assert.True(t, errors.Is(<-runtimeErr, ErrProjectManagerClosed))
	runtime := <-created
	assert.False(t, runtime.Events.Publish("closed"))
	_, err := manager.Runtime(context.Background(), project)
	assert.True(t, errors.Is(err, ErrProjectManagerClosed))
	assert.NoErr(t, manager.Close())
}
