package utils

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

var  EnableDebug bool

func Debugf(format string, args ...any) {
	if !EnableDebug {
		return
	}
	clog.Debugf("[DEBUG] "+format, args...)
}

func FormatTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

func FormatFileSize(size int64) string {
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

func IsMarkdownFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".md")
}

func IsMarkdownFilePresent(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && IsMarkdownFile(info.Name())
}

func NormalizeRelativePath(path string) string {
	return filepath.ToSlash(path)
}

func ToURLPath(relativePath string) string {
	normalized := NormalizeRelativePath(relativePath)
	if normalized == "" || normalized == "." {
		return "/"
	}

	segments := strings.Split(normalized, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return "/" + strings.Join(segments, "/")
}

func MustMarshalJSON(value any) template.JS {
	payload, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return template.JS(payload)
}
