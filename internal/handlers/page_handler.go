package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var (
	customTagOpenPattern   = regexp.MustCompile(`^<([A-Za-z][A-Za-z0-9_-]*)(\s[^<>]*)?>\s*$`)
	customTagClosePattern  = regexp.MustCompile(`^</([A-Za-z][A-Za-z0-9_-]*)>\s*$`)
	customTagAttrPattern   = regexp.MustCompile(`([A-Za-z_:][A-Za-z0-9_:.-]*)\s*=\s*("([^"]*)"|'([^']*)'|([^\s"'>/]+))`)
	customTagInlinePattern = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9_-]*(?:\s[^<>]*)?>`)
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
				goldhtml.WithHardWraps(),
				goldhtml.WithUnsafe(), // Allow raw HTML
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
	AppConfigJSON       template.JS
	CurrentFilePath     string
	AppVersion          string
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
			renderDirectoryListing(w, r, filePath)
			return
		}
	}

	utils.Debugf("Request: %s query: %s, filePath: %s", urlPath, r.URL.Query(), filePath)

	// Render markdown file
	if strings.HasSuffix(strings.ToLower(filePath), ".md") {
		queryParam := r.URL.Query().Get("q")
		switch queryParam {
		case queryTypeMain: // 只渲染 markdown 内容，不带其他部分
			renderMainContent(w, filePath)
		case queryTypeRaw: // 直接返回原始 markdown 内容
			renderRawMarkdown(w, filePath)
		default: // 返回完整 html 页面
			renderMarkdown(w, filePath)
		}
		return
	}

	// Serve static file
	http.ServeFile(w, r, filePath)
}

func renderMarkdown(w http.ResponseWriter, filePath string) {
	mainData, err := buildPageData(filePath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	renderFullPage(w, mainData)
}

func renderFullPage(w http.ResponseWriter, mainData *PageData) {
	targetDir := config.Cfg.TargetDir

	fileTree, err := buildFileTree(targetDir)
	if err != nil {
		http.Error(w, "Failed to build file tree", 500)
		return
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
		Title:         mainData.FileName,
		MainContent:   template.HTML(mainContentBuf.String()),
		FileTreeJSON:  utils.MustMarshalJSON(fileTree),
		AppConfigJSON: utils.MustMarshalJSON(config.Cfg.AppConfig()),
		AppVersion:    config.Cfg.Version,
	}

	setPageCacheHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, data)
}

func renderDirectoryListing(w http.ResponseWriter, r *http.Request, dirPath string) {
	mainData, rawMarkdown, err := buildDirectoryListingPageData(dirPath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	queryParam := r.URL.Query().Get("q")
	if queryParam == queryTypeMain {
		renderPageMainContent(w, mainData)
		return
	}
	if queryParam == queryTypeRaw {
		setPageCacheHeaders(w)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(rawMarkdown))
		return
	}

	renderFullPage(w, mainData)
}

// renderMarkdownContent renders just the markdown content (shared function)
func renderMarkdownContent(filePath string) (string, error) {
	mdData, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return renderMarkdownSource(mdData)
}

func renderMarkdownSource(mdData []byte) (string, error) {
	initMdParser()
	var buf bytes.Buffer
	if err := mdParser.Convert(normalizeCustomMarkdownTags(mdData), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type customTag struct {
	name       string
	attrs      map[string]string
	isClosing  bool
	isStandard bool
}

func parseCustomHTMLTag(raw string) (customTag, bool) {
	trimmed := strings.TrimSpace(raw)
	if matches := customTagClosePattern.FindStringSubmatch(trimmed); matches != nil {
		name := strings.ToLower(matches[1])
		return customTag{name: name, isClosing: true, isStandard: isStandardHTMLTag(name)}, true
	}

	matches := customTagOpenPattern.FindStringSubmatch(trimmed)
	if matches == nil {
		return customTag{}, false
	}

	name := strings.ToLower(matches[1])
	return customTag{
		name:       name,
		attrs:      parseCustomTagAttrs(matches[2]),
		isStandard: isStandardHTMLTag(name),
	}, true
}

func normalizeCustomMarkdownTags(mdData []byte) []byte {
	lines := strings.SplitAfter(string(mdData), "\n")
	var builder strings.Builder
	builder.Grow(len(mdData))
	inFence := false

	for _, line := range lines {
		lineBody := strings.TrimRight(line, "\r\n")
		fenceMarker := strings.TrimLeft(strings.TrimSpace(lineBody), " \t")
		if strings.HasPrefix(fenceMarker, "```") || strings.HasPrefix(fenceMarker, "~~~") {
			inFence = !inFence
			builder.WriteString(line)
			continue
		}

		if inFence {
			builder.WriteString(line)
			continue
		}

		if writeCustomBlockTagLine(&builder, lineBody) {
			continue
		}

		builder.WriteString(rewriteInlineCustomTags(lineBody))
		if strings.HasSuffix(line, "\n") {
			builder.WriteByte('\n')
		}
	}

	return []byte(builder.String())
}

func writeCustomBlockTagLine(builder *strings.Builder, line string) bool {
	tag, ok := parseCustomHTMLTag(line)
	if !ok || tag.isStandard {
		return false
	}

	if tag.isClosing {
		builder.WriteByte('\n')
		builder.WriteString("</div>\n")
		return true
	}

	builder.WriteString(renderCustomTagOpen(tag, "div"))
	builder.WriteString("\n\n")
	return true
}

func rewriteInlineCustomTags(line string) string {
	matches := customTagInlinePattern.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	codeRanges := inlineCodeRanges(line)
	var builder strings.Builder
	builder.Grow(len(line))
	last := 0
	changed := false

	for _, match := range matches {
		if rangeContainsIndex(codeRanges, match[0]) {
			continue
		}

		raw := line[match[0]:match[1]]
		tag, ok := parseCustomHTMLTag(raw)
		if !ok || tag.isStandard {
			continue
		}

		builder.WriteString(line[last:match[0]])
		if tag.isClosing {
			builder.WriteString("</span>")
		} else {
			builder.WriteString(renderCustomTagOpen(tag, "span"))
		}
		last = match[1]
		changed = true
	}

	if !changed {
		return line
	}

	builder.WriteString(line[last:])
	return builder.String()
}

func inlineCodeRanges(line string) [][2]int {
	ranges := make([][2]int, 0)
	for pos := 0; pos < len(line); {
		if line[pos] != '`' {
			pos++
			continue
		}

		tickCount := countBackticks(line, pos)
		end := findClosingBackticks(line, pos+tickCount, tickCount)
		if end == -1 {
			pos += tickCount
			continue
		}

		ranges = append(ranges, [2]int{pos, end + tickCount})
		pos = end + tickCount
	}
	return ranges
}

func countBackticks(line string, pos int) int {
	count := 0
	for pos+count < len(line) && line[pos+count] == '`' {
		count++
	}
	return count
}

func findClosingBackticks(line string, start int, tickCount int) int {
	for pos := start; pos < len(line); pos++ {
		if line[pos] != '`' {
			continue
		}
		if countBackticks(line, pos) == tickCount {
			return pos
		}
	}
	return -1
}

func rangeContainsIndex(ranges [][2]int, index int) bool {
	for _, value := range ranges {
		if index >= value[0] && index < value[1] {
			return true
		}
	}
	return false
}

func renderCustomTagOpen(tag customTag, element string) string {
	var builder strings.Builder
	_, _ = fmt.Fprintf(
		&builder,
		`<%s class="markdown-custom-tag markdown-custom-tag-%s" data-markview-tag="%s"`,
		element,
		stdhtml.EscapeString(tag.name),
		stdhtml.EscapeString(tag.name),
	)

	for _, attr := range sortedCustomTagAttrs(tag.attrs) {
		_, _ = fmt.Fprintf(&builder, ` data-markview-attr-%s="%s"`, attr.name, stdhtml.EscapeString(attr.value))
	}
	builder.WriteByte('>')
	return builder.String()
}

func parseCustomTagAttrs(raw string) map[string]string {
	attrs := map[string]string{}
	for _, match := range customTagAttrPattern.FindAllStringSubmatch(raw, -1) {
		name := normalizeCustomTagAttrName(match[1])
		if name == "" {
			continue
		}

		value := match[3]
		if value == "" {
			value = match[4]
		}
		if value == "" {
			value = match[5]
		}
		attrs[name] = value
	}

	return attrs
}

type customTagAttr struct {
	name  string
	value string
}

func sortedCustomTagAttrs(attrs map[string]string) []customTagAttr {
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]customTagAttr, 0, len(names))
	for _, name := range names {
		result = append(result, customTagAttr{name: name, value: attrs[name]})
	}
	return result
}

func normalizeCustomTagAttrName(name string) string {
	name = strings.ToLower(name)
	var builder strings.Builder
	lastDash := false
	for _, char := range name {
		isAlphaNum := char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
		if isAlphaNum {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func isStandardHTMLTag(name string) bool {
	switch name {
	case "a", "abbr", "address", "area", "article", "aside", "audio", "b", "base", "bdi", "bdo",
		"blockquote", "body", "br", "button", "canvas", "caption", "cite", "code", "col",
		"colgroup", "data", "datalist", "dd", "del", "details", "dfn", "dialog", "div", "dl",
		"dt", "em", "embed", "fieldset", "figcaption", "figure", "footer", "form", "h1", "h2",
		"h3", "h4", "h5", "h6", "head", "header", "hr", "html", "i", "iframe", "img", "input",
		"ins", "kbd", "label", "legend", "li", "link", "main", "map", "mark", "menu", "meta",
		"meter", "nav", "noscript", "object", "ol", "optgroup", "option", "output", "p",
		"picture", "pre", "progress", "q", "rp", "rt", "ruby", "s", "samp", "script", "search",
		"section", "select", "slot", "small", "source", "span", "strong", "style", "sub",
		"summary", "sup", "svg", "table", "tbody", "td", "template", "textarea", "tfoot",
		"th", "thead", "time", "title", "tr", "track", "u", "ul", "var", "video", "wbr":
		return true
	default:
		return false
	}
}

func renderRawMarkdown(w http.ResponseWriter, filePath string) {
	mdData, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	setPageCacheHeaders(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(mdData)
}

func renderMainContent(w http.ResponseWriter, filePath string) {
	mainData, err := buildPageData(filePath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	renderPageMainContent(w, mainData)
}

func renderPageMainContent(w http.ResponseWriter, mainData *PageData) {
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(mainContentBuf.Bytes())
}

// buildPageData 提取公共的页面数据构建逻辑
func buildPageData(filePath string) (*PageData, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to stat file: %v", err)
	}

	contentHTML, err := renderMarkdownContent(filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to render markdown: %v", err)
	}

	fileName := filepath.Base(filePath)

	createdAt := "Unavailable"
	if created := utils.FileCreatedTime(info); !created.IsZero() {
		createdAt = utils.FormatTimestamp(created)
	}

	targetDir := config.Cfg.TargetDir
	currentRelativePath, err := filepath.Rel(targetDir, filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve current file path: %v", err)
	}

	return &PageData{
		Title:               fileName,
		Content:             template.HTML(contentHTML),
		FileName:            fileName,
		FilePath:            filePath,
		FileSize:            utils.FormatFileSize(info.Size()),
		CreatedAt:           createdAt,
		ModifiedAt:          utils.FormatTimestamp(info.ModTime()),
		CurrentFilePathJSON: utils.MustMarshalJSON(utils.NormalizeRelativePath(currentRelativePath)),
		CurrentFilePath:     utils.ToURLPath(currentRelativePath),
	}, nil
}

func buildDirectoryListingPageData(dirPath string) (*PageData, string, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to stat directory: %v", err)
	}

	targetDir := config.Cfg.TargetDir
	currentRelativePath, err := filepath.Rel(targetDir, dirPath)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to resolve current directory path: %v", err)
	}
	currentRelativePath = utils.NormalizeRelativePath(currentRelativePath)
	if currentRelativePath == "." {
		currentRelativePath = ""
	}

	rawMarkdown, err := buildDirectoryListingMarkdown(dirPath, currentRelativePath)
	if err != nil {
		return nil, "", err
	}
	contentHTML, err := renderMarkdownSource([]byte(rawMarkdown))
	if err != nil {
		return nil, "", fmt.Errorf("Failed to render directory listing: %v", err)
	}

	fileName := filepath.Base(dirPath)
	if currentRelativePath == "" {
		fileName = filepath.Base(targetDir)
	}
	createdAt := "Unavailable"
	if created := utils.FileCreatedTime(info); !created.IsZero() {
		createdAt = utils.FormatTimestamp(created)
	}

	return &PageData{
		Title:               fileName,
		Content:             template.HTML(contentHTML),
		FileName:            fileName,
		FilePath:            dirPath,
		FileSize:            "Directory",
		CreatedAt:           createdAt,
		ModifiedAt:          utils.FormatTimestamp(info.ModTime()),
		CurrentFilePathJSON: utils.MustMarshalJSON(currentRelativePath),
		CurrentFilePath:     utils.ToURLPath(currentRelativePath),
	}, rawMarkdown, nil
}

func buildDirectoryListingMarkdown(dirPath, relativeDir string) (string, error) {
	nodes, err := buildFileTreeDir(dirPath, relativeDir)
	if err != nil {
		return "", fmt.Errorf("Failed to build directory listing: %v", err)
	}

	title := filepath.Base(dirPath)
	var builder strings.Builder
	builder.WriteString("# 📇 ")
	builder.WriteString(escapeMarkdownText(title))
	builder.WriteString("\n\n")

	if len(nodes) == 0 {
		builder.WriteString("No related files.\n")
		return builder.String(), nil
	}

	for _, node := range nodes {
		label := node.Name
		if node.Kind == "directory" {
			label += "/"
		}
		// 使用绝对站内链接，避免前端按当前目录再次重写相对路径。
		builder.WriteString("- [")
		builder.WriteString(escapeMarkdownLinkText(label))
		builder.WriteString("](")
		builder.WriteString(node.Href)
		builder.WriteString(")\n")
	}

	return builder.String(), nil
}

func escapeMarkdownText(text string) string {
	return strings.NewReplacer(`\`, `\\`, `#`, `\#`).Replace(text)
}

func escapeMarkdownLinkText(text string) string {
	return strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`).Replace(text)
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
				Navigable: true,
				Href:      utils.ToURLPath(entryRelativePath),
				Children:  children,
			}
			if hasIndex {
				node.MatchPath = utils.NormalizeRelativePath(filepath.Join(entryRelativePath, "index.md"))
			} else {
				node.MatchPath = utils.NormalizeRelativePath(entryRelativePath)
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
	utils.Debugf("Request: %s handle file tree", r.URL.Path)
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
