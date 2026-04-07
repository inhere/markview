package handlers

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/utils"
)

var watcher *fsnotify.Watcher

var (
	debounceTimer *time.Timer
	debounceMutex sync.Mutex
	pendingFiles  = make(map[string]bool) // key: path, value: isNew
	watchedDir    string
)

// ReloadMessage for JSON notification format
type ReloadMessage struct {
	Type   string   `json:"type"`
	Files  []string `json:"files"`
	Action string   `json:"action,omitempty"`
}

// WatchDirectory watches the directory and its subdirectories for changes.
func WatchDirectory(dir string) {
	watchedDir = dir

	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()
	watchDirs := config.Cfg.WatchDirs

	// Walk and add all subdirectories
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			// Skip directories start with dot or in watchSkipDirs
			if shouldSkipDir(name) {
				utils.Debugf("Skip watch directory: %s", path)
				return filepath.SkipDir
			}
			// watchDirs is not empty, only watch directories in watchDirs
			if len(watchDirs) > 0 && !slices.Contains(watchDirs, name) {
				utils.Debugf("Skip watch directory: %s", path)
				return filepath.SkipDir
			}

			utils.Debugf("Watch directory: %s", path)
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		clog.Error("Error walking directory:", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				// event.Name is full path
				if strings.HasSuffix(event.Name, ".md") {
					relPath, _ := filepath.Rel(dir, event.Name)
					handleFileChange(relPath, event)
				}
			}
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil {
					if info.IsDir() {
						watcher.Add(event.Name)
					} else if strings.HasSuffix(event.Name, ".md") {
						relPath, _ := filepath.Rel(dir, event.Name)
						handleFileChange(relPath, event)
					}
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			clog.Errorf("WATCH: Watcher error: %v", err)
		}
	}
}

func handleFileChange(filePath string, event fsnotify.Event) {
	debounceMutex.Lock()
	defer debounceMutex.Unlock()

	isNew := event.Has(fsnotify.Create)
	if pendingFiles == nil {
		pendingFiles = make(map[string]bool)
	}

	if existingIsNew, exists := pendingFiles[filePath]; !exists {
		pendingFiles[filePath] = isNew
		clog.Infof("Modified file: %s (%s)", filePath, event.Op.String())
	} else if isNew && !existingIsNew {
		pendingFiles[filePath] = true
	}

	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	debounceTimer = time.AfterFunc(2*time.Second, func() {
		debounceMutex.Lock()
		files := pendingFiles
		pendingFiles = make(map[string]bool)
		debounceMutex.Unlock()

		if len(files) > 0 {
			broadcastJSON(files)
		}
	})
}

func broadcastJSON(files map[string]bool) {
	hasCreate := false
	paths := make([]string, 0, len(files))
	for path, isNew := range files {
		paths = append(paths, path)
		if isNew {
			hasCreate = true
		}
	}

	msg := ReloadMessage{
		Type:  "reload",
		Files: paths,
	}
	if hasCreate {
		msg.Action = "create"
	}
	data, err := json.Marshal(msg)
	if err != nil {
		clog.Errorf("Failed to marshal reload message: %v", err)
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		select {
		case client <- string(data):
		default:
		}
	}
}
