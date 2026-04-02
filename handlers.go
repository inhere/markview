package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type FileTreeNode struct {
	Name      string         `json:"name"`
	Href      string         `json:"href,omitempty"`
	MatchPath string         `json:"matchPath,omitempty"`
	Kind      string         `json:"kind"`
	Navigable bool           `json:"navigable"`
	Children  []FileTreeNode `json:"children,omitempty"`
}

// handleRequest .md 文件会渲染为 HTML 页面，其他文件会直接返回
func handleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	var filePath, cleanPath string

	if urlPath == "/" {
		cleanPath = defaultEntry
		filePath = filepath.Join(targetDir, defaultEntry)
	} else {
		// Remove leading slash
		cleanPath = strings.TrimPrefix(urlPath, "/")
		filePath = filepath.Join(targetDir, cleanPath)
	}

	// Security check
	rel, err := filepath.Rel(targetDir, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		http.Error(w, "Access Denied", 403)
		return
	}

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.Error(w, "File Not Found: "+cleanPath, 404)
		return
	}

	if info.IsDir() {
		// Serve index.md if exists in directory
		indexPath := filepath.Join(filePath, "index.md")
		if _, err := os.Stat(indexPath); err == nil {
			filePath = indexPath
		} else {
			// List directory? For now just 404
			http.Error(w, "Directory listing not supported", 404)
			return
		}
	}

	debugf("Request: %s query: %s, filePath: %s", urlPath, r.URL.Query(), filePath)

		// Render markdown file
	if strings.HasSuffix(strings.ToLower(filePath), ".md") {
		queryParam := r.URL.Query().Get("q")
		if queryParam == "main" {
			renderMainContent(w, filePath)
		} else {
			renderMarkdown(w, filePath)
		}
		return
	}

	// Serve static file
	http.ServeFile(w, r, filePath)
}

func renderMarkdown(w http.ResponseWriter, filePath string) {
	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "Failed to stat file", 500)
		return
	}

	contentHTML, err := renderMarkdownContent(filePath)
	if err != nil {
		http.Error(w, "Failed to render markdown", 500)
		return
	}

	fileName := filepath.Base(filePath)
	createdAt := "Unavailable"
	if created := fileCreatedTime(info); !created.IsZero() {
		createdAt = formatTimestamp(created)
	}
	currentRelativePath, err := filepath.Rel(targetDir, filePath)
	if err != nil {
		http.Error(w, "Failed to resolve current file path", 500)
		return
	}

	fileTree, err := buildFileTree(targetDir)
	if err != nil {
		http.Error(w, "Failed to build file tree", 500)
		return
	}

	mainData := PageData{
		Title:               fileName,
		Content:             template.HTML(contentHTML),
		FileName:            fileName,
		FilePath:            filePath,
		FileSize:            formatFileSize(info.Size()),
		CreatedAt:           createdAt,
		ModifiedAt:          formatTimestamp(info.ModTime()),
		CurrentFilePathJSON: mustMarshalJSON(normalizeRelativePath(currentRelativePath)),
	}

	mainTmplData, err := content.ReadFile("web/template-main.html")
	if err != nil {
		http.Error(w, "Template-main not found", 500)
		return
	}
	mainTmpl := template.Must(template.New("main").Parse(string(mainTmplData)))
	var mainContentBuf bytes.Buffer
	if err = mainTmpl.Execute(&mainContentBuf, mainData); err != nil {
		http.Error(w, "Failed to render main template", 500)
		return
	}

	tmplData, err := content.ReadFile("web/template.html")
	if err != nil {
		http.Error(w, "Template not found", 500)
		return
	}
	t := template.Must(template.New("index").Parse(string(tmplData)))

	data := PageData{
		Title:        fileName,
		MainContent:  template.HTML(mainContentBuf.String()),
		FileTreeJSON: mustMarshalJSON(fileTree),
		// CurrentFilePathJSON: mustMarshalJSON(normalizeRelativePath(currentRelativePath)),
	}

	setPageCacheHeaders(w)
	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

// renderMarkdownContent renders just the markdown content (shared function)
func renderMarkdownContent(filePath string) (string, error) {
	mdData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Configure goldmark
	md := goldmark.New(
		// GFM 扩展支持表格、删除线、链接化和任务列表
		goldmark.WithExtensions(extension.GFM, emoji.Emoji, meta.New(meta.WithTable())),
		goldmark.WithParserOptions(
		// parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithUnsafe(), // Allow raw HTML
			renderer.WithNodeRenderers(
				util.Prioritized(extension.NewTableHTMLRenderer(), 500),
			),
		),
	)

	var buf bytes.Buffer
	if err = md.Convert(mdData, &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func renderMainContent(w http.ResponseWriter, filePath string) {
	info, err := os.Stat(filePath)
	if err != nil {
		http.Error(w, "Failed to stat file", 500)
		return
	}

	contentHTML, err := renderMarkdownContent(filePath)
	if err != nil {
		http.Error(w, "Failed to render markdown", 500)
		return
	}

	fileName := filepath.Base(filePath)
	createdAt := "Unavailable"
	if created := fileCreatedTime(info); !created.IsZero() {
		createdAt = formatTimestamp(created)
	}
	currentRelativePath, err := filepath.Rel(targetDir, filePath)
	if err != nil {
		http.Error(w, "Failed to resolve current file path", 500)
		return
	}

	mainData := PageData{
		Title:               fileName,
		Content:             template.HTML(contentHTML),
		FileName:            fileName,
		FilePath:            filePath,
		FileSize:            formatFileSize(info.Size()),
		CreatedAt:           createdAt,
		ModifiedAt:          formatTimestamp(info.ModTime()),
		CurrentFilePathJSON: mustMarshalJSON(normalizeRelativePath(currentRelativePath)),
	}

	mainTmplData, err := content.ReadFile("web/template-main.html")
	if err != nil {
		http.Error(w, "Template-main not found", 500)
		return
	}
	mainTmpl := template.Must(template.New("main").Parse(string(mainTmplData)))
	var mainContentBuf bytes.Buffer
	if err = mainTmpl.Execute(&mainContentBuf, mainData); err != nil {
		http.Error(w, "Failed to render main template", 500)
		return
	}

	setPageCacheHeaders(w)
	w.Header().Set("Content-Type", "text/html")
	w.Write(mainContentBuf.Bytes())
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan string)

	clientsMu.Lock()
	clients[clientChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, clientChan)
		clientsMu.Unlock()
		close(clientChan)
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-time.After(15 * time.Second):
			// Keep alive
			fmt.Fprintf(w, ": keepalive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

func buildFileTree(root string) ([]FileTreeNode, error) {
	return buildFileTreeDir(root, "")
}

func buildFileTreeDir(absDir, relativeDir string) ([]FileTreeNode, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}

	directories := make([]FileTreeNode, 0)
	files := make([]FileTreeNode, 0)

	for _, entry := range entries {
		if entry.IsDir() && shouldSkipDir(entry.Name()) {
			continue
		}

		entryRelativePath := entry.Name()
		if relativeDir != "" {
			entryRelativePath = filepath.Join(relativeDir, entry.Name())
		}
		entryAbsolutePath := filepath.Join(absDir, entry.Name())

		if entry.IsDir() {
			children, err := buildFileTreeDir(entryAbsolutePath, entryRelativePath)
			if err != nil {
				return nil, err
			}

			indexPath := filepath.Join(entryAbsolutePath, "index.md")
			hasIndex := isMarkdownFilePresent(indexPath)
			if !hasIndex && len(children) == 0 {
				continue
			}

			node := FileTreeNode{
				Name:      entry.Name(),
				Kind:      "directory",
				Navigable: hasIndex,
				Children:  children,
			}
			if hasIndex {
				node.Href = toURLPath(entryRelativePath)
				node.MatchPath = normalizeRelativePath(filepath.Join(entryRelativePath, "index.md"))
			}

			directories = append(directories, node)
			continue
		}

		if !isMarkdownFile(entry.Name()) {
			continue
		}
		if relativeDir != "" && strings.EqualFold(entry.Name(), "index.md") {
			continue
		}

		normalizedPath := normalizeRelativePath(entryRelativePath)
		files = append(files, FileTreeNode{
			Name:      entry.Name(),
			Href:      toURLPath(normalizedPath),
			MatchPath: normalizedPath,
			Kind:      "file",
			Navigable: true,
		})
	}

	sortFileTreeNodes(directories)
	sortFileTreeNodes(files)

	return append(directories, files...), nil
}

func sortFileTreeNodes(nodes []FileTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		left := strings.ToLower(nodes[i].Name)
		right := strings.ToLower(nodes[j].Name)
		if left == right {
			return nodes[i].Name < nodes[j].Name
		}
		return left < right
	})
}

// Skip directories start with dot or in watchSkipDirs
func shouldSkipDir(name string) bool {
	// Skip directories start with dot
	if name[0] == '.' {
		return true
	}
	return slices.Contains(watchSkipDirs, name)
}

func setPageCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

func setStaticCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
}
