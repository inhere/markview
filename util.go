package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gookit/goutil/x/clog"
)

func debugf(format string, args ...any) {
	if !enableDebug {
		return
	}
	clog.Debugf(format, args...)
}

func formatTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func isMarkdownFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".md")
}

func isMarkdownFilePresent(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && isMarkdownFile(info.Name())
}

func normalizeRelativePath(path string) string {
	return filepath.ToSlash(path)
}

func toURLPath(relativePath string) string {
	normalized := normalizeRelativePath(relativePath)
	if normalized == "" || normalized == "." {
		return "/"
	}

	segments := strings.Split(normalized, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return "/" + strings.Join(segments, "/")
}

func mustMarshalJSON(value any) template.JS {
	payload, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return template.JS(payload)
}
