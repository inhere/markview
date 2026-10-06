//go:build !windows

package handlers

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/inhere/markview/internal/config"
)

// fsnotifyWatcher 使用 fsnotify 监听项目树，为每个目录单独注册监听。
type fsnotifyWatcher struct {
	root    ProjectRoot
	cfg     config.Config
	native  *fsnotify.Watcher
	changes chan nativeChange
	errs    chan error
	done    chan struct{}
	once    sync.Once
}

func newNativeWatcher(root ProjectRoot, cfg config.Config) (nativeWatcher, error) {
	native, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := &fsnotifyWatcher{
		root:    root,
		cfg:     cfg,
		native:  native,
		changes: make(chan nativeChange, 32),
		errs:    make(chan error, 4),
		done:    make(chan struct{}),
	}
	if err := watcher.addInitialDirectories(); err != nil {
		native.Close()
		return nil, err
	}
	go watcher.loop()
	return watcher, nil
}

func (w *fsnotifyWatcher) Events() <-chan nativeChange { return w.changes }
func (w *fsnotifyWatcher) Errors() <-chan error        { return w.errs }

func (w *fsnotifyWatcher) Close() error {
	var err error
	w.once.Do(func() {
		close(w.done)
		err = w.native.Close()
	})
	return err
}

func (w *fsnotifyWatcher) loop() {
	defer close(w.changes)
	defer close(w.errs)
	for {
		select {
		case <-w.done:
			return
		case err, ok := <-w.native.Errors:
			if !ok {
				return
			}
			select {
			case w.errs <- err:
			case <-w.done:
			}
			return
		case event, ok := <-w.native.Events:
			if !ok {
				return
			}
			change, notify := w.handleEvent(event)
			if !notify {
				continue
			}
			select {
			case w.changes <- change:
			case <-w.done:
				return
			}
		}
	}
}

func (w *fsnotifyWatcher) addInitialDirectories() error {
	return filepath.WalkDir(w.root.DisplayPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(w.root.DisplayPath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return w.native.Add(w.root.RealPath)
		}
		if skipWatchDirectory(rel, entry.Name(), w.cfg) {
			return filepath.SkipDir
		}
		resolved, err := w.root.Resolve("/" + filepath.ToSlash(rel))
		if errors.Is(err, ErrPathOutsideProject) {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}
		return w.native.Add(resolved)
	})
}

func (w *fsnotifyWatcher) handleEvent(event fsnotify.Event) (nativeChange, bool) {
	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err != nil {
			return nativeChange{}, false
		}
		if info.IsDir() {
			resolved, err := w.resolveEventPath(event.Name)
			rel, relErr := filepath.Rel(w.root.RealPath, event.Name)
			if err == nil && relErr == nil && !skipWatchDirectory(rel, info.Name(), w.cfg) {
				_ = w.native.Add(resolved)
			}
			return nativeChange{}, false
		}
		return w.relativeMarkdownChange(event.Name, EventTypeCreate)
	}
	if event.Has(fsnotify.Write) {
		return w.relativeMarkdownChange(event.Name, EventTypeUpdate)
	}
	return nativeChange{}, false
}

func (w *fsnotifyWatcher) relativeMarkdownChange(path, eventType string) (nativeChange, bool) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return nativeChange{}, false
	}
	resolved, err := w.resolveEventPath(path)
	if err != nil {
		return nativeChange{}, false
	}
	rel, err := filepath.Rel(w.root.RealPath, resolved)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nativeChange{}, false
	}
	return nativeChange{Rel: filepath.ToSlash(rel), Op: eventType}, true
}

func (w *fsnotifyWatcher) resolveEventPath(path string) (string, error) {
	rel, err := filepath.Rel(w.root.RealPath, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathOutsideProject
	}
	return w.root.Resolve("/" + filepath.ToSlash(rel))
}
