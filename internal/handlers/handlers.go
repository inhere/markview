package handlers

import (
	"net/http"
	"slices"

	"github.com/inhere/markview/internal/config"
)

// Skip directories start with dot or in watchSkipDirs
func shouldSkipDir(name string) bool {
	return shouldSkipDirForConfig(name, config.Cfg)
}

func shouldSkipDirForConfig(name string, cfg config.Config) bool {
	if name == ".git" || name == "node_modules" {
		return true
	}
	if slices.Contains(cfg.IncludeDirs, name) {
		return false
	}
	// Skip directories start with dot
	if name[0] == '.' {
		return true
	}
	return slices.Contains(cfg.WatchSkipDirs, name)
}

func setPageCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}
