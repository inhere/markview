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
	pendingFiles  []string
	watchedDir    string
)

// ReloadMessage for JSON notification format
type ReloadMessage struct {
	Type  string   `json:"type"`
	Files []string `json:"files"`
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
				if err == nil && info.IsDir() {
					watcher.Add(event.Name)
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

	if !slices.Contains(pendingFiles, filePath) {
		pendingFiles = append(pendingFiles, filePath)
		clog.Infof("Modified file: %s (%s)", filePath, event.Op.String())
	}

	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	debounceTimer = time.AfterFunc(2*time.Second, func() {
		debounceMutex.Lock()
		files := make([]string, len(pendingFiles))
		copy(files, pendingFiles)
		pendingFiles = nil
		debounceMutex.Unlock()

		if len(files) > 0 {
			broadcastJSON(files)
		}
	})
}

func broadcastJSON(files []string) {
	msg := ReloadMessage{
		Type:  "reload",
		Files: files,
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
