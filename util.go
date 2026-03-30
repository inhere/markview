package main

import (
	"encoding/json"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

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
