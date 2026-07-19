package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/utils"
)

const (
	EventTypeUpdate = "update"
	EventTypeCreate = "create"
)

type ReloadMessage struct {
	Type   string   `json:"type"`
	Files  []string `json:"files"`
	Action string   `json:"action,omitempty"`
}

type Watcher struct {
	root      ProjectRoot
	config    config.Config
	events    *EventHub
	native    *fsnotify.Watcher
	debounce  time.Duration
	done      chan struct{}
	closeOnce sync.Once
}

func NewWatcher(root ProjectRoot, cfg config.Config, events *EventHub) (*Watcher, error) {
	native, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	watcher := &Watcher{
		root:     root,
		config:   cfg,
		events:   events,
		native:   native,
		debounce: 1500 * time.Millisecond,
		done:     make(chan struct{}),
	}
	if err := watcher.addInitialDirectories(); err != nil {
		native.Close()
		return nil, err
	}
	return watcher, nil
}

func (watcher *Watcher) Run(ctx context.Context) error {
	pending := make(map[string]string)
	var timer *time.Timer
	var timerC <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-watcher.done:
			return nil
		case err, ok := <-watcher.native.Errors:
			if !ok {
				return nil
			}
			return err
		case event, ok := <-watcher.native.Events:
			if !ok {
				return nil
			}
			path, eventType, notify := watcher.handleEvent(event)
			if !notify {
				continue
			}
			pending[path] = eventType
			if timer == nil {
				timer = time.NewTimer(watcher.debounce)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(watcher.debounce)
			}
			timerC = timer.C
		case <-timerC:
			watcher.publish(pending)
			pending = make(map[string]string)
			timerC = nil
		}
	}
}

func (watcher *Watcher) Close() error {
	var err error
	watcher.closeOnce.Do(func() {
		close(watcher.done)
		err = watcher.native.Close()
	})
	return err
}

func (watcher *Watcher) addInitialDirectories() error {
	return filepath.WalkDir(watcher.root.DisplayPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(watcher.root.DisplayPath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return watcher.native.Add(watcher.root.RealPath)
		}
		if watcher.skipDirectory(rel, entry.Name()) {
			return filepath.SkipDir
		}
		resolved, err := watcher.root.Resolve("/" + filepath.ToSlash(rel))
		if errors.Is(err, ErrPathOutsideProject) {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}
		return watcher.native.Add(resolved)
	})
}

func (watcher *Watcher) skipDirectory(relativePath, name string) bool {
	if shouldSkipDirForConfig(name, watcher.config) {
		return true
	}
	if len(watcher.config.WatchDirs) == 0 {
		return false
	}
	first := strings.Split(filepath.ToSlash(relativePath), "/")[0]
	return !slices.Contains(watcher.config.WatchDirs, first)
}

func (watcher *Watcher) handleEvent(event fsnotify.Event) (string, string, bool) {
	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err != nil {
			return "", "", false
		}
		if info.IsDir() {
			resolved, err := watcher.resolveEventPath(event.Name)
			rel, relErr := filepath.Rel(watcher.root.RealPath, event.Name)
			if err == nil && relErr == nil && !watcher.skipDirectory(rel, info.Name()) {
				_ = watcher.native.Add(resolved)
			}
			return "", "", false
		}
		return watcher.relativeMarkdownPath(event.Name, EventTypeCreate)
	}
	if event.Has(fsnotify.Write) {
		return watcher.relativeMarkdownPath(event.Name, EventTypeUpdate)
	}
	return "", "", false
}

func (watcher *Watcher) relativeMarkdownPath(path, eventType string) (string, string, bool) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return "", "", false
	}
	resolved, err := watcher.resolveEventPath(path)
	if err != nil {
		return "", "", false
	}
	rel, err := filepath.Rel(watcher.root.RealPath, resolved)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", false
	}
	return filepath.ToSlash(rel), eventType, true
}

func (watcher *Watcher) resolveEventPath(path string) (string, error) {
	rel, err := filepath.Rel(watcher.root.RealPath, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathOutsideProject
	}
	return watcher.root.Resolve("/" + filepath.ToSlash(rel))
}

func (watcher *Watcher) publish(files map[string]string) {
	if len(files) == 0 {
		return
	}
	paths := make([]string, 0, len(files))
	action := EventTypeUpdate
	for path, eventType := range files {
		paths = append(paths, path)
		if eventType == EventTypeCreate {
			action = EventTypeCreate
		}
	}
	sort.Strings(paths)
	data, err := json.Marshal(ReloadMessage{Type: "reload", Files: paths, Action: action})
	if err != nil {
		clog.Errorf("Failed to marshal reload message: %v", err)
		return
	}
	watcher.events.Publish(string(data))
}

func WatchDirectory(dir string) {
	root, err := NewProjectRoot(dir)
	if err != nil {
		clog.Errorf("WATCH: resolve project root: %v", err)
		return
	}
	watcher, err := NewWatcher(root, config.Cfg, defaultEventHub)
	if err != nil {
		clog.Errorf("WATCH: create watcher: %v", err)
		return
	}
	defer watcher.Close()
	utils.Debugf("Watching project %s", root.DisplayPath)
	if err := watcher.Run(context.Background()); err != nil {
		clog.Errorf("WATCH: %v", err)
	}
}
