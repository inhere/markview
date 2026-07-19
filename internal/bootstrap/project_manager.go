package bootstrap

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/handlers"
	"github.com/inhere/markview/internal/projects"
)

var (
	ErrProjectManagerClosed = errors.New("project manager closed")
	ErrProjectUnavailable   = errors.New("project unavailable")
)

type ProjectRuntime struct {
	ID      string
	Path    string
	Config  config.Config
	Server  *handlers.ProjectServer
	Watcher *handlers.Watcher
	Events  *handlers.EventHub

	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

func (runtime *ProjectRuntime) Close() error {
	runtime.closeOnce.Do(func() {
		if runtime.cancel != nil {
			runtime.cancel()
		}
		var watcherErr, eventErr error
		if runtime.Watcher != nil {
			watcherErr = runtime.Watcher.Close()
		}
		if runtime.Events != nil {
			eventErr = runtime.Events.Close()
		}
		runtime.closeErr = errors.Join(watcherErr, eventErr)
	})
	return runtime.closeErr
}

type runtimeSlot struct {
	ready   chan struct{}
	runtime *ProjectRuntime
	err     error
}

type ProjectManager struct {
	mu       sync.Mutex
	closed   bool
	runtimes map[string]*ProjectRuntime
	loading  map[string]*runtimeSlot
	inFlight sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	factory  func(context.Context, projects.IndexedProject) (*ProjectRuntime, error)
	content  fs.FS

	closeOnce sync.Once
	closeErr  error
}

func NewProjectManager(content fs.FS) *ProjectManager {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &ProjectManager{
		runtimes: make(map[string]*ProjectRuntime),
		loading:  make(map[string]*runtimeSlot),
		ctx:      ctx,
		cancel:   cancel,
		content:  content,
	}
	manager.factory = func(ctx context.Context, project projects.IndexedProject) (*ProjectRuntime, error) {
		return buildProjectRuntime(ctx, project, content)
	}
	return manager
}

func (manager *ProjectManager) Runtime(ctx context.Context, project projects.IndexedProject) (*ProjectRuntime, error) {
	if !project.Exists {
		return nil, ErrProjectUnavailable
	}
	manager.mu.Lock()
	if manager.closed {
		manager.mu.Unlock()
		return nil, ErrProjectManagerClosed
	}
	if runtime := manager.runtimes[project.ID]; runtime != nil {
		manager.mu.Unlock()
		return runtime, nil
	}
	if slot := manager.loading[project.ID]; slot != nil {
		manager.mu.Unlock()
		return waitRuntimeSlot(ctx, slot)
	}
	slot := &runtimeSlot{ready: make(chan struct{})}
	manager.loading[project.ID] = slot
	manager.inFlight.Add(1)
	manager.mu.Unlock()

	runtime, err := manager.factory(manager.ctx, project)
	manager.finishRuntime(project.ID, slot, runtime, err)
	return waitRuntimeSlot(ctx, slot)
}

func (manager *ProjectManager) finishRuntime(id string, slot *runtimeSlot, runtime *ProjectRuntime, initErr error) {
	manager.mu.Lock()
	closed := manager.closed
	delete(manager.loading, id)
	if !closed && initErr == nil {
		manager.runtimes[id] = runtime
	}
	manager.mu.Unlock()
	if closed {
		if runtime != nil {
			_ = runtime.Close()
			runtime = nil
		}
		initErr = ErrProjectManagerClosed
	}

	manager.mu.Lock()
	slot.runtime = runtime
	slot.err = initErr
	close(slot.ready)
	manager.mu.Unlock()
	manager.inFlight.Done()
}

func waitRuntimeSlot(ctx context.Context, slot *runtimeSlot) (*ProjectRuntime, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-slot.ready:
		return slot.runtime, slot.err
	}
}

func (manager *ProjectManager) Close() error {
	manager.closeOnce.Do(func() {
		manager.mu.Lock()
		manager.closed = true
		manager.cancel()
		manager.mu.Unlock()
		manager.inFlight.Wait()

		manager.mu.Lock()
		runtimes := make([]*ProjectRuntime, 0, len(manager.runtimes))
		for id, runtime := range manager.runtimes {
			runtimes = append(runtimes, runtime)
			delete(manager.runtimes, id)
		}
		manager.mu.Unlock()
		var errs []error
		for _, runtime := range runtimes {
			errs = append(errs, runtime.Close())
		}
		manager.closeErr = errors.Join(errs...)
	})
	return manager.closeErr
}

func buildProjectRuntime(ctx context.Context, project projects.IndexedProject, content fs.FS) (*ProjectRuntime, error) {
	cfg, err := config.LoadProjectRuntimeConfig(project.Path, config.ProjectLoadOptions{GlobalMode: true})
	if err != nil {
		return nil, err
	}
	if err := cfg.Init(project.Path, ""); err != nil {
		return nil, err
	}
	cfg.BasePath = "/p/" + project.ID
	cfg.ProjectName = project.Record.Name
	if cfg.ProjectName == "" {
		cfg.ProjectName = filepath.Base(project.Path)
	}
	cfg.ProjectPath = simplifyProjectPath(project.Path)
	root, err := handlers.NewProjectRoot(project.Path)
	if err != nil {
		return nil, err
	}
	events := handlers.NewEventHub()
	runtimeCtx, cancel := context.WithCancel(ctx)
	runtime := &ProjectRuntime{
		ID:     project.ID,
		Path:   project.Path,
		Config: cfg,
		Events: events,
		cancel: cancel,
	}
	runtime.Server = handlers.NewProjectServer(cfg, root, events, content)
	if !cfg.EnableWatch {
		return runtime, nil
	}
	runtime.Watcher, err = handlers.NewWatcher(root, cfg, events)
	if err != nil {
		_ = runtime.Close()
		return nil, err
	}
	go func() {
		if err := runtime.Watcher.Run(runtimeCtx); err != nil && !errors.Is(err, context.Canceled) {
			clog.Errorf("WATCH %s: %v", project.Path, err)
		}
	}()
	return runtime, nil
}
