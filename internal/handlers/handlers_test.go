package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inhere/markview/internal/config"
)

func TestBuildFileTree(t *testing.T) {
	root := t.TempDir()

	files := []string{
		"README.md",
		"notes.txt",
		"guide/index.md",
		"guide/intro.md",
		"guide/images/logo.png",
		"guide/deep/topic.md",
		"guide/deep/index.md",
		"plain/child.md",
		"z-last.md",
	}

	for _, relativePath := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", relativePath, err)
		}
		if err := os.WriteFile(fullPath, []byte("# test"), 0o644); err != nil {
			t.Fatalf("write %s: %v", relativePath, err)
		}
	}

	tree, err := buildFileTree(root)
	if err != nil {
		t.Fatalf("buildFileTree returned error: %v", err)
	}

	if len(tree) != 4 {
		t.Fatalf("expected 4 root nodes, got %d", len(tree))
	}

	if tree[0].Kind != "directory" || tree[0].Name != "guide" {
		t.Fatalf("expected first node to be guide directory, got %+v", tree[0])
	}
	if !tree[0].Navigable {
		t.Fatalf("expected guide directory to be navigable")
	}
	if tree[0].Href != "/guide" {
		t.Fatalf("expected guide href /guide, got %s", tree[0].Href)
	}
	if tree[0].MatchPath != "guide/index.md" {
		t.Fatalf("expected guide matchPath guide/index.md, got %s", tree[0].MatchPath)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("expected guide to expose 2 children, got %d", len(tree[0].Children))
	}
	if tree[0].Children[0].Kind != "directory" || tree[0].Children[0].Name != "deep" {
		t.Fatalf("expected guide child 0 to be deep directory, got %+v", tree[0].Children[0])
	}
	if tree[0].Children[1].Kind != "file" || tree[0].Children[1].Name != "intro.md" {
		t.Fatalf("expected guide child 1 to be intro.md, got %+v", tree[0].Children[1])
	}

	if tree[1].Kind != "directory" || tree[1].Name != "plain" {
		t.Fatalf("expected second node to be plain directory, got %+v", tree[1])
	}
	if tree[1].Navigable {
		t.Fatalf("expected plain directory to be non-navigable without index.md")
	}
	if len(tree[1].Children) != 1 || tree[1].Children[0].Name != "child.md" {
		t.Fatalf("expected plain to contain child.md, got %+v", tree[1].Children)
	}

	if tree[2].Kind != "file" || tree[2].Name != "README.md" {
		t.Fatalf("expected third node to be README.md, got %+v", tree[2])
	}
	if tree[3].Kind != "file" || tree[3].Name != "z-last.md" {
		t.Fatalf("expected fourth node to be z-last.md, got %+v", tree[3])
	}
}

func TestHandleRequestSetsNoStoreForMarkdownPages(t *testing.T) {
	IfsReader = func(path string) ([]byte, error) {
		if path == "web/template-main.html" {
			return []byte(`{{.Content}}`), nil
		}
		if path == "web/template.html" {
			return []byte(`<html>{{.MainContent}}</html>`), nil
		}
		return nil, os.ErrNotExist
	}

	root := t.TempDir()
	config.Cfg.TargetDir = root
	config.Cfg.EntryFile = "README.md"

	readmePath := filepath.Join(root, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Hello"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	HandleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", got)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected html content-type, got %q", contentType)
	}
}

// --- 搜索功能回归测试：文件级 AND/NOT 语义 ---
// 参考：https://github.com/inhere/markview/issues/XXX
// 期望语义：
//   - 文件级 AND：所有 include 关键词分布在整个文件中即可匹配（不要求在同一行）
//   - 文件级 NOT：任意位置有 exclude 关键词就排除整个文件

func TestParseSearchTerms(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantInc []string
		wantExc []string
	}{
		{
			name:    "single include",
			query:   "hello",
			wantInc: []string{"hello"},
			wantExc: nil,
		},
		{
			name:    "multiple include (AND)",
			query:   "hello world",
			wantInc: []string{"hello", "world"},
			wantExc: nil,
		},
		{
			name:    "single exclude",
			query:   "!draft",
			wantInc: nil,
			wantExc: []string{"draft"},
		},
		{
			name:    "mixed include and exclude",
			query:   "api !deprecated",
			wantInc: []string{"api"},
			wantExc: []string{"deprecated"},
		},
		{
			name:    "multiple exclude",
			query:   "test !draft !wip",
			wantInc: []string{"test"},
			wantExc: []string{"draft", "wip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSearchTerms(tt.query)
			if len(got.Include) != len(tt.wantInc) {
				t.Errorf("Include: got %v, want %v", got.Include, tt.wantInc)
			}
			if len(got.Exclude) != len(tt.wantExc) {
				t.Errorf("Exclude: got %v, want %v", got.Exclude, tt.wantExc)
			}
			// 验证 lowercase 版本
			if len(got.includeLower) != len(tt.wantInc) {
				t.Errorf("includeLower: got %v, want %v", got.includeLower, tt.wantInc)
			}
		})
	}
}

// TestLineMatchesMatch_IncludeAcrossLines 测试文件级 AND 语义：
// 多个 include 关键词分布在不同行时，整个文件应该匹配
func TestLineMatchesMatch_IncludeAcrossLines(t *testing.T) {
	tests := []struct {
		name        string
		content     string // 模拟多行内容
		query       string
		wantMatch   bool // 期望是否匹配
		description string
	}{
		{
			name: "include 关键词在同一行 - 应该匹配",
			content: `# API Guide

This document covers the API endpoints.
`,
			query:       "API endpoints",
			wantMatch:   true,
			description: "同一行包含所有 include 词",
		},
		{
			name: "include 关键词分布在不同行 - 当前实现：期望匹配（文件级AND）",
			content: `# API Guide

This document covers the API.

It also mentions endpoints.
`,
			query:       "API endpoints",
			wantMatch:   true, // 文件级 AND：两词分布在不同行也应该匹配
			description: "不同行包含不同 include 词",
		},
		{
			name: "只有部分 include 关键词存在 - 当前实现：不匹配",
			content: `# API Guide

This document covers the API only.
`,
			query:       "API endpoints",
			wantMatch:   false, // "endpoints" 不存在
			description: "缺少一个 include 词",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := parseSearchTerms(tt.query)
			lines := strings.Split(tt.content, "\n")

			// 测试文件级逻辑：逐行判断，但包含文件级 NOT 上下文
			var matchedLines []int
			for i, line := range lines {
				// 传入文件内容以支持文件级 NOT 检查
				if lineMatchesMatch(line, terms, tt.content) {
					matchedLines = append(matchedLines, i+1)
				}
			}

			// 文件级 AND 语义：只要有匹配行即文件匹配
			hasMatch := len(matchedLines) > 0
			if hasMatch != tt.wantMatch {
				t.Errorf("case: %s\ncontent:\n%s\nquery: %s\nmatched lines: %v\ngot match=%v, want match=%v",
					tt.name, tt.content, tt.query, matchedLines, hasMatch, tt.wantMatch)
			}
		})
	}
}

// TestLineMatchesMatch_ExcludeFileLevel 测试文件级 NOT 语义：
// exclude 关键词出现在文件任意位置时，整个文件应该被排除
func TestLineMatchesMatch_ExcludeFileLevel(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		query       string
		wantMatch   bool // 文件级 NOT：只要有 exclude 就不匹配
		description string
	}{
		{
			name: "exclude 词在第一行 - 应该排除文件",
			content: `# TODO: this is draft

This is the API documentation.
`,
			query:       "API !draft",
			wantMatch:   false, // 文件级 NOT：任意位置有 "draft" 就排除
			description: "exclude 词在开头行",
		},
		{
			name: "exclude 词在最后一行 - 应该排除文件",
			content: `# API Documentation

This is the API content.

Status: draft
`,
			query:       "API !draft",
			wantMatch:   false, // 文件级 NOT
			description: "exclude 词在末尾行",
		},
		{
			name: "exclude 词在中间行 - 应该排除文件",
			content: `# API Guide

This is the API.

[mark draft]

More content here.
`,
			query:       "API !draft",
			wantMatch:   false, // 文件级 NOT
			description: "exclude 词在中间行",
		},
		{
			name: "include 词匹配但 exclude 在不同行 - 应该排除文件",
			content: `# API Guide

This is about API.

Also mentions draft somewhere else.
`,
			query:       "API !draft",
			wantMatch:   false, // 文件级 NOT：即使 API 匹配，draft 存在也要排除
			description: "include 匹配但 exclude 存在",
		},
		{
			name: "无 exclude 词 - 正常匹配",
			content: `# API Guide

This is the API documentation.
`,
			query:       "API",
			wantMatch:   true,
			description: "无 exclude 词",
		},
		{
			name: "只有 exclude 词（无 include）- 文件级 NOT 应排除",
			content: `# API Guide

This is draft content.
`,
			query:       "!draft",
			wantMatch:   false, // 文件级 NOT：有 exclude 词就排除
			description: "仅 exclude 查询",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := parseSearchTerms(tt.query)
			lines := strings.Split(tt.content, "\n")

			// 文件级逻辑：逐行判断，但包含文件级 NOT 上下文
			var matchedLines []int
			for i, line := range lines {
				// 传入文件内容以支持文件级 NOT 检查
				if lineMatchesMatch(line, terms, tt.content) {
					matchedLines = append(matchedLines, i+1)
				}
			}

			hasMatch := len(matchedLines) > 0
			if hasMatch != tt.wantMatch {
				t.Errorf("case: %s\ncontent:\n%s\nquery: %s\nmatched lines: %v\ngot match=%v, want match=%v",
					tt.name, tt.content, tt.query, matchedLines, hasMatch, tt.wantMatch)
			}
		})
	}
}

// TestSearchInContent_PureExclude 纯 exclude 查询的回归测试
// 新语义：纯 exclude 查询只保留文件命中，不返回行级 matches
// 期望：matches 应为空数组（不返回每行的匹配）
func TestSearchInContent_PureExclude(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		query     string
		wantMatch bool   // 期望文件是否匹配（未被排除）
		wantCount int    // 期望的 matches 数量（新语义：纯 exclude 应返回 0）
		reason    string // 说明原因
	}{
		{
			name: "纯 exclude 查询 - 文件不含 exclude 关键词：新语义返回 0 条 matches",
			content: `# API Guide

This is the API documentation.
It covers all endpoints.
`,
			query:     "!draft",
			wantMatch: true, // 文件未被排除，应该命中
			wantCount: 0,    // 新语义：纯 exclude 不返回行级 matches
			reason:    "纯 exclude 查询只保留文件命中，不返回行级 matches",
		},
		{
			name: "纯 exclude 查询 - 文件含 exclude 关键词：文件被排除",
			content: `# API Guide

This is draft content.
`,
			query:     "!draft",
			wantMatch: false, // 文件含 draft，被排除
			wantCount: 0,     // 文件被排除，返回 0 条
			reason:    "文件含 exclude 关键词，整个文件被排除",
		},
		{
			name: "纯 exclude 查询 - 多行内容无 exclude 词",
			content: `# API Guide

Line 1 content
Line 2 content
Line 3 content
Line 4 content
Line 5 content
`,
			query:     "!deprecated",
			wantMatch: true, // 文件未被排除，应该命中
			wantCount: 0,    // 新语义：纯 exclude 不返回行级 matches
			reason:    "纯 exclude 查询只保留文件命中",
		},
		{
			name: "纯 exclude 查询 - 多个 exclude 词但文件都不含",
			content: `# API Guide

This is the API documentation.
`,
			query:     "!draft !wip !archived",
			wantMatch: true, // 都不含，文件命中
			wantCount: 0,    // 新语义
			reason:    "多个 exclude 词但文件都不含",
		},
		{
			name: "纯 exclude 查询 - 多个 exclude 词但文件含其中一个",
			content: `# API Guide

This is wip content.
`,
			query:     "!draft !wip !archived",
			wantMatch: false, // 含 wip，被排除
			wantCount: 0,
			reason:    "含其中一个 exclude 词即排除",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := parseSearchTerms(tt.query)
			matches := searchInContent(tt.content, terms, 30)

			// 新语义下：纯 exclude 查询的验证逻辑
			// wantMatch=true + wantCount=0 表示文件命中但无行级 matches
			// wantMatch=false + wantCount=0 表示文件被排除
			if len(matches) != tt.wantCount {
				t.Errorf("case: %s\ncontent:\n%s\nquery: %s\ngot %d matches, want %d\nreason: %s",
					tt.name, tt.content, tt.query, len(matches), tt.wantCount, tt.reason)
			}
		})
	}
}

// TestSearchInContent_FileLevelSemantic 集成测试：searchInContent 的文件级语义
func TestSearchInContent_FileLevelSemantic(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		query     string
		wantCount int // 期望的匹配行数
	}{
		{
			name: "文件级 AND 语义：include 分布不同行应返回匹配行",
			content: `# API Guide
This is the API.
It mentions endpoints here.`,
			query:     "API endpoints",
			wantCount: 1, // 文件级 AND：文件同时包含 API 和 endpoints，通过后返回匹配行
		},
		{
			name: "文件级 NOT 语义：exclude 存在时整文件排除",
			content: `# API Guide
This is the API.
Status: draft
More content.`,
			query:     "API !draft",
			wantCount: 0, // 文件级 NOT：文件包含 draft，整个文件被排除
		},
		{
			name: "文件级 NOT 语义：exclude 存在应返回空",
			content: `# API Guide
This is the API.
Status: draft`,
			query:     "API !draft",
			wantCount: 0, // 文件级 NOT：整个文件应被排除，返回 0 行
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := parseSearchTerms(tt.query)
			matches := searchInContent(tt.content, terms, 30)

			if len(matches) != tt.wantCount {
				t.Errorf("case: %s\ncontent:\n%s\nquery: %s\ngot %d matches, want %d\nmatches: %v",
					tt.name, tt.content, tt.query, len(matches), tt.wantCount, matches)
			}
		})
	}
}

// TestHandleSearch_PureExcludeQuery 端到端测试：纯 exclude 查询应返回命中文件（即使 matches 为空）
// 期望语义：
//   - 纯 exclude 查询命中不含排除词的文件时，响应 results 非空
//   - 对应的 result.matches 为空数组 []
//   - total 为 0（因为没有实际匹配行）
//
// 失败原因：HandleSearch 中 `if len(matches) > 0` 导致 len(matches)==0 的文件被过滤掉
func TestHandleSearch_PureExcludeQuery(t *testing.T) {
	// 创建临时目录和测试文件
	root := t.TempDir()

	// 创建一个不含 "draft" 的文件（纯 exclude 查询应该命中它）
	cleanFile := filepath.Join(root, "clean-doc.md")
	cleanContent := `# API Guide

This is the API documentation.
It covers all endpoints.
`
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0o644); err != nil {
		t.Fatalf("write clean-doc.md: %v", err)
	}

	// 创建一个含 "draft" 的文件（纯 exclude 查询应该排除它）
	draftFile := filepath.Join(root, "draft-doc.md")
	draftContent := `# Draft Guide

This is draft content.
`
	if err := os.WriteFile(draftFile, []byte(draftContent), 0o644); err != nil {
		t.Fatalf("write draft-doc.md: %v", err)
	}

	// 配置使用临时目录
	origTargetDir := config.Cfg.TargetDir
	config.Cfg.TargetDir = root
	defer func() { config.Cfg.TargetDir = origTargetDir }()

	// 发起纯 exclude 查询：!draft（查找所有不含 draft 的文件）
	req := httptest.NewRequest(http.MethodGet, "/search?q=!draft", nil)
	rec := httptest.NewRecorder()

	HandleSearch(rec, req)

	// 验证响应
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// 断言 1：results 非空（clean-doc.md 应该命中）
	if len(resp.Results) == 0 {
		t.Fatalf("FAIL: 纯 exclude 查询命中不含排除词的文件时，results 应非空\n" +
			"期望：至少有 1 个文件（clean-doc.md）\n" +
			"实际：results 为空\n" +
			"失败原因：HandleSearch 第 79 行 `if len(matches) > 0` 把 len(matches)==0 的文件过滤掉了\n" +
			"searchInContent 对纯 exclude 返回 []SearchMatch{}，len 为 0，导致文件不进入 results")
	}

	// 断言 2：找到 clean-doc.md
	var cleanResult *SearchResult
	for i := range resp.Results {
		if strings.Contains(resp.Results[i].File, "clean-doc.md") {
			cleanResult = &resp.Results[i]
			break
		}
	}
	if cleanResult == nil {
		t.Fatalf("FAIL: 应找到 clean-doc.md\n实际 results: %v", resp.Results)
	}

	// 断言 3：matches 应为空数组（不是 nil，是空切片 []）
	if cleanResult.Matches == nil {
		t.Errorf("FAIL: matches 不应为 nil，应为空切片 []\n期望：matches = []\n实际：matches = nil")
	} else if len(cleanResult.Matches) != 0 {
		t.Errorf("FAIL: matches 长度应为 0\n期望：len(matches) = 0\n实际：len(matches) = %d", len(cleanResult.Matches))
	}

	// 断言 4：draft-doc.md 不应在结果中（被排除）
	for i := range resp.Results {
		if strings.Contains(resp.Results[i].File, "draft-doc.md") {
			t.Errorf("FAIL: draft-doc.md 不应出现在结果中（被 exclude 排除）")
		}
	}

	// 断言 5：total 应为 0（没有实际匹配行，只有文件命中）
	if resp.Total != 0 {
		t.Errorf("FAIL: total 应为 0（纯 exclude 只有文件命中，无行级匹配）\n期望：total = 0\n实际：total = %d", resp.Total)
	}

	t.Logf("PASS: 纯 exclude 查询语义正确\nresults count: %d\nclean-doc.md matches: %v\ntotal: %d",
		len(resp.Results), cleanResult.Matches, resp.Total)
}
