package handlers

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/utils"
)

var watcher *fsnotify.Watcher

// WatchDirectory watches the directory and its subdirectories for changes.
func WatchDirectory(dir string) {
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
				if strings.HasSuffix(event.Name, ".md") {
					relPath, _ := filepath.Rel(dir, event.Name)
					clog.Warnf("Modified file: %s(event: %s)", relPath, event.String())
					broadcast("reload")
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

func broadcast(msg string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		select {
		case client <- msg:
		default:
			// Client blocked, ignore
		}
	}
}
