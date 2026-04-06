package handlers

import (
	"net/http"
	"slices"

	"github.com/inhere/markview/internal/config"
)

// Skip directories start with dot or in watchSkipDirs
func shouldSkipDir(name string) bool {
	// Skip directories start with dot
	if name[0] == '.' {
		return true
	}
	return slices.Contains(config.Cfg.WatchSkipDirs, name)
}

func setPageCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}
