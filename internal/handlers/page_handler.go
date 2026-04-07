package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/utils"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// 全局 goldmark Markdown 解析器单例 (线程安全)
var mdParser goldmark.Markdown
var initMdParserOnce sync.Once

// initMdParser 初始化全局 markdown 解析器 (只执行一次)
func initMdParser() {
	initMdParserOnce.Do(func() {
		mdParser = goldmark.New(
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
	})
}

type PageData struct {
	Title               string
	Content             template.HTML
	MainContent         template.HTML
	FileName            string
	FilePath            string
	FileSize            string
	CreatedAt           string
	ModifiedAt          string
	FileTreeJSON        template.JS
	CurrentFilePathJSON template.JS
}

// IfsReader 从 embed.FS 读取文件内容
var IfsReader func(path string) ([]byte, error)

// HandleRequest .md 文件会渲染为 HTML 页面，其他文件会直接返回
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	targetDir := config.Cfg.TargetDir
	var filePath, cleanPath string

	if urlPath == "/" {
		cleanPath = config.Cfg.EntryFile
		filePath = filepath.Join(targetDir, cleanPath)
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

	utils.Debugf("Request: %s query: %s, filePath: %s", urlPath, r.URL.Query(), filePath)

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

	targetDir := config.Cfg.TargetDir
	contentHTML, err := renderMarkdownContent(filePath)
	if err != nil {
		http.Error(w, "Failed to render markdown", 500)
		return
	}

	fileName := filepath.Base(filePath)
	createdAt := "Unavailable"
	if created := utils.FileCreatedTime(info); !created.IsZero() {
		createdAt = utils.FormatTimestamp(created)
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
		FileSize:            utils.FormatFileSize(info.Size()),
		CreatedAt:           createdAt,
		ModifiedAt:          utils.FormatTimestamp(info.ModTime()),
		CurrentFilePathJSON: utils.MustMarshalJSON(utils.NormalizeRelativePath(currentRelativePath)),
	}

	mainTmplData, err := IfsReader("web/template-main.html")
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

	tmplData, err := IfsReader("web/template.html")
	if err != nil {
		http.Error(w, "Template not found", 500)
		return
	}
	t := template.Must(template.New("index").Parse(string(tmplData)))

	data := PageData{
		Title:        fileName,
		MainContent:  template.HTML(mainContentBuf.String()),
		FileTreeJSON: utils.MustMarshalJSON(fileTree),
		// CurrentFilePathJSON: utils.MustMarshalJSON(normalizeRelativePath(currentRelativePath)),
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

	initMdParser()
	var buf bytes.Buffer
	if err = mdParser.Convert(mdData, &buf); err != nil {
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
	if created := utils.FileCreatedTime(info); !created.IsZero() {
		createdAt = utils.FormatTimestamp(created)
	}
	currentRelativePath, err := filepath.Rel(config.Cfg.TargetDir, filePath)
	if err != nil {
		http.Error(w, "Failed to resolve current file path", 500)
		return
	}

	mainData := PageData{
		Title:               fileName,
		Content:             template.HTML(contentHTML),
		FileName:            fileName,
		FilePath:            filePath,
		FileSize:            utils.FormatFileSize(info.Size()),
		CreatedAt:           createdAt,
		ModifiedAt:          utils.FormatTimestamp(info.ModTime()),
		CurrentFilePathJSON: utils.MustMarshalJSON(utils.NormalizeRelativePath(currentRelativePath)),
	}

	mainTmplData, err := IfsReader("web/template-main.html")
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

type FileTreeNode struct {
	Name      string         `json:"name"`
	Href      string         `json:"href,omitempty"`
	MatchPath string         `json:"matchPath,omitempty"`
	Kind      string         `json:"kind"`
	Navigable bool           `json:"navigable"`
	Children  []FileTreeNode `json:"children,omitempty"`
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
			hasIndex := utils.IsMarkdownFilePresent(indexPath)
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
				node.Href = utils.ToURLPath(entryRelativePath)
				node.MatchPath = utils.NormalizeRelativePath(filepath.Join(entryRelativePath, "index.md"))
			}

			directories = append(directories, node)
			continue
		}

		if !utils.IsMarkdownFile(entry.Name()) {
			continue
		}
		if relativeDir != "" && strings.EqualFold(entry.Name(), "index.md") {
			continue
		}

		normalizedPath := utils.NormalizeRelativePath(entryRelativePath)
		files = append(files, FileTreeNode{
			Name:      entry.Name(),
			Href:      utils.ToURLPath(normalizedPath),
			MatchPath: normalizedPath,
			Kind:      "file",
			Navigable: true,
		})
	}

	sortFileTreeNodes(directories)
	sortFileTreeNodes(files)

	return append(directories, files...), nil
}

// HandleFileTreeAPI returns file tree JSON for dynamic refresh
func HandleFileTreeAPI(w http.ResponseWriter, r *http.Request) {
	targetDir := config.Cfg.TargetDir
	fileTree, err := buildFileTree(targetDir)
	if err != nil {
		http.Error(w, "Failed to build file tree", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	data, err := json.Marshal(fileTree)
	if err != nil {
		http.Error(w, "Failed to marshal file tree", 500)
		return
	}
	w.Write(data)
}

func sortFileTreeNodes(nodes []FileTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		nameI, nameJ := nodes[i].Name, nodes[j].Name
		// 使用 EqualFold 进行大小写不敏感比较
		if strings.EqualFold(nameI, nameJ) {
			return nameI < nameJ
		}
		// 按小写比较实现大小写不敏感排序
		return strings.ToLower(nameI) < strings.ToLower(nameJ)
	})
}
