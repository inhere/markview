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
	pendingFiles  []fileChange
	watchedDir    string
)

type fileChange struct {
	path  string
	isNew bool
}

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
						clog.Infof("Created file: %s", relPath)
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
	for i, f := range pendingFiles {
		if f.path == filePath {
			if isNew {
				pendingFiles[i].isNew = true
			}
			return
		}
	}
	pendingFiles = append(pendingFiles, fileChange{path: filePath, isNew: isNew})

	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	debounceTimer = time.AfterFunc(2*time.Second, func() {
		debounceMutex.Lock()
		files := make([]fileChange, len(pendingFiles))
		copy(files, pendingFiles)
		pendingFiles = nil
		debounceMutex.Unlock()

		if len(files) > 0 {
			broadcastJSON(files)
		}
	})
}

func broadcastJSON(files []fileChange) {
	hasCreate := false
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.path
		if f.isNew {
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
