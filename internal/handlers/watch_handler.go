package handlers

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/utils"
)

var watcher *fsnotify.Watcher

// channel-based debounce architecture
// 使用 channel 架构替代全局 mutex + timer
var (
	watchedDir  string
	eventChan   = make(chan string, 100) // buffered channel for file change events
	stopChan    = make(chan struct{})
	debounceDur = 200 * time.Millisecond // 200ms for faster live reloads
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

	// 启动防抖处理 goroutine
	go debounceProcessor()

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

	// 等待 stop 信号
	for {
		select {
		case <-stopChan:
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				// event.Name is full path
				if strings.HasSuffix(event.Name, ".md") {
					relPath, _ := filepath.Rel(dir, event.Name)
					handleFileChange(relPath)
				}
			}
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil {
					if info.IsDir() {
						watcher.Add(event.Name)
					} else if strings.HasSuffix(event.Name, ".md") {
						relPath, _ := filepath.Rel(dir, event.Name)
						handleFileChange(relPath)
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

// handleFileChange 简化为只向 channel 发送事件，不涉及任何锁操作
// 线程安全：所有状态管理都在 debounceProcessor goroutine 中处理
func handleFileChange(filePath string) {
	select {
	case eventChan <- filePath:
		clog.Debugf("File change queued: %s", filePath)
	default:
		// channel 满了说明事件堆积，跳过这次
		clog.Warnf("Event channel full, dropping: %s", filePath)
	}
}

// debounceProcessor 使用 timer 进行防抖处理
// 这是一个独立的 goroutine，集中管理所有状态，保证线程安全
func debounceProcessor() {
	var timer *time.Timer
	files := make(map[string]bool) // 收集待处理文件

	for {
		select {
		case <-stopChan:
			// 清理 timer
			if timer != nil {
				timer.Stop()
			}
			return

		case filePath := <-eventChan:
			// 收集文件变更
			files[filePath] = true
			clog.Debugf("File event received: %s (pending: %d)", filePath, len(files))

			// 重置 timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(debounceDur)

		case <-func() <-chan time.Time {
			if timer != nil {
				return timer.C
			}
			return nil
		}():
			// Timer 触发时，在这个主 goroutine 中读取 files
			if len(files) > 0 {
				pendingFiles := files
				files = make(map[string]bool)

				clog.Infof("Broadcasting %d file changes", len(pendingFiles))
				// broadcastJSON 执行较快且内部有锁，直接调用即可
				broadcastJSON(pendingFiles)
			}
		}
	}
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
