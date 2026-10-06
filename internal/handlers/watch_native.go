package handlers

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/inhere/markview/internal/config"
)

// nativeChange 是平台 watcher 上报的原始 markdown 变更。
type nativeChange struct {
	// Rel 是相对项目根目录、以 / 分隔的路径。
	Rel string
	// Op 为 EventTypeCreate 或 EventTypeUpdate。
	Op string
}

// nativeWatcher 抽象平台文件监听后端：Windows 用单个递归句柄，
// 其它平台用 fsnotify。改动的 markdown 通过 Events 上报。
type nativeWatcher interface {
	Events() <-chan nativeChange
	Errors() <-chan error
	Close() error
}

// skipWatchDirectory 判断目录是否应排除在监听范围之外。
func skipWatchDirectory(relativePath, name string, cfg config.Config) bool {
	if shouldSkipDirForConfig(name, cfg) {
		return true
	}
	if len(cfg.WatchDirs) == 0 {
		return false
	}
	first := strings.Split(filepath.ToSlash(relativePath), "/")[0]
	return !slices.Contains(cfg.WatchDirs, first)
}

// markdownChangeFromRel 校验相对路径并转换为变更事件，过滤非 markdown 文件
// 以及处于跳过目录（.git、node_modules 等）中的路径。
func markdownChangeFromRel(rel, op string, cfg config.Config) (nativeChange, bool) {
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return nativeChange{}, false
	}
	if !strings.EqualFold(filepath.Ext(rel), ".md") {
		return nativeChange{}, false
	}

	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return nativeChange{Rel: rel, Op: op}, true
	}

	segments := strings.Split(dir, "/")
	if len(cfg.WatchDirs) > 0 && !slices.Contains(cfg.WatchDirs, segments[0]) {
		return nativeChange{}, false
	}
	for _, name := range segments {
		if shouldSkipDirForConfig(name, cfg) {
			return nativeChange{}, false
		}
	}
	return nativeChange{Rel: rel, Op: op}, true
}
