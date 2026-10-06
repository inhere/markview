package handlers

import (
	"context"
	"encoding/json"
	"sort"
	"time"

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
	native   nativeWatcher
	events   *EventHub
	debounce time.Duration
}

func NewWatcher(root ProjectRoot, cfg config.Config, events *EventHub) (*Watcher, error) {
	native, err := newNativeWatcher(root, cfg)
	if err != nil {
		return nil, err
	}
	return &Watcher{
		native:   native,
		events:   events,
		debounce: 1500 * time.Millisecond,
	}, nil
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
		case err, ok := <-watcher.native.Errors():
			if !ok {
				return nil
			}
			return err
		case change, ok := <-watcher.native.Events():
			if !ok {
				return nil
			}
			pending[change.Rel] = change.Op
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
	return watcher.native.Close()
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
