package handlers

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"github.com/inhere/markview/internal/utils"
)

// HandleSearch 搜索 API handler
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResponse{
			Query:        "",
			Results:      []SearchResult{},
			Total:        0,
			Duration:     0,
			FilesScanned: 0,
		})
		return
	}

	start := time.Now()
	terms := parseSearchTerms(query)
	results := []SearchResult{}
	maxFiles := 15
	maxMatchesPerFile := 30
	filesScanned := 0
	targetDir := config.Cfg.TargetDir

	// Walk through targetDir to find .md files
	err := filepath.WalkDir(targetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			clog.Error("Failed to walk dir: %v", err)
			return nil
		}

		// Skip directories
		if d.IsDir() {
			name := d.Name()
			if shouldSkipDir(name) {
				utils.Debugf("Search: Skipping dir: %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Skip index.md in subdirectories
		relPath, _ := filepath.Rel(targetDir, path)
		if strings.Contains(filepath.Dir(relPath), string(filepath.Separator)) &&
			strings.EqualFold(filepath.Base(path), "index.md") {
			return nil
		}

		filesScanned++

		// Read and search file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		matches := searchInContent(string(content), terms, maxMatchesPerFile)
		if len(matches) > 0 {
			results = append(results, SearchResult{
				File:    relPath,
				Matches: matches,
			})
		}

		if len(results) >= maxFiles {
			return errors.New("max files limit reached")
		}

		return nil
	})

	if err != nil && err.Error() != "max files limit reached" {
		http.Error(w, "Search error: "+err.Error(), 500)
		return
	}

	duration := int(time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SearchResponse{
		Query:        query,
		Results:      results,
		Total:        countMatches(results),
		Duration:     duration,
		FilesScanned: filesScanned,
	})
}

// SearchTerms 解析后的搜索条件
type SearchTerms struct {
	Include      []string // 原始包含关键词
	Exclude      []string // 原始排除关键词
	IncludeLower []string // 预计算的 lowercase 版本，避免在搜索循环中重复转换
	ExcludeLower []string // 预计算的 lowercase 版本
}

// SearchMatch 匹配的行
type SearchMatch struct {
	Line    int      `json:"line"`
	Snippet string   `json:"snippet"`
	Lines   []int    `json:"lines,omitempty"`
	Context []string `json:"context,omitempty"`
}

// SearchResult 单个文件的搜索结果
type SearchResult struct {
	File    string        `json:"file"`
	Matches []SearchMatch `json:"matches"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Query        string         `json:"query"`
	Results      []SearchResult `json:"results"`
	Total        int            `json:"total"`
	Duration     int            `json:"duration"`
	FilesScanned int            `json:"filesScanned"`
}

// parseSearchTerms 解析查询字符串
// 空格 = AND，! 前缀 = 排除
// 预计算 IncludeLower 和 ExcludeLower 以减少搜索循环中的字符串分配
func parseSearchTerms(query string) SearchTerms {
	terms := SearchTerms{
		Include:      []string{},
		Exclude:      []string{},
		IncludeLower: []string{},
		ExcludeLower: []string{},
	}

	words := strings.Fields(query)
	for _, word := range words {
		if strings.HasPrefix(word, "!") {
			cleanWord := strings.TrimPrefix(word, "!")
			terms.Exclude = append(terms.Exclude, cleanWord)
			terms.ExcludeLower = append(terms.ExcludeLower, strings.ToLower(cleanWord))
		} else {
			terms.Include = append(terms.Include, word)
			terms.IncludeLower = append(terms.IncludeLower, strings.ToLower(word))
		}
	}

	return terms
}

// lineMatchesMatch 检查行是否匹配搜索条件
// 使用预计算的 ExcludeLower 和 IncludeLower 避免重复的 ToLower 调用
func lineMatchesMatch(line string, terms SearchTerms) bool {
	lineLower := strings.ToLower(line)

	// Check exclude terms first
	for _, ex := range terms.ExcludeLower {
		if strings.Contains(lineLower, ex) {
			return false
		}
	}

	// If no include terms, match all (for exclude-only queries)
	if len(terms.Include) == 0 {
		return true
	}

	// Check include terms (AND logic)
	for _, inc := range terms.IncludeLower {
		if !strings.Contains(lineLower, inc) {
			return false
		}
	}

	return true
}

// searchInContent 在内容中搜索匹配行，收集上下文
func searchInContent(content string, terms SearchTerms, limit int) []SearchMatch {
	lines := strings.Split(content, "\n")
	matches := []SearchMatch{}
	skipUntil := -1 // 跳过到的行号

	for i, line := range lines {
		// 跳过已经作为上下文收集的行
		if i < skipUntil {
			continue
		}

		if lineMatchesMatch(line, terms) {
			// 收集上下文：上一行 + current + 后两行
			startLine := max(0, i-1)
			endLine := min(len(lines)-1, i+2)

			contextLines := []string{}
			lineNums := []int{}
			for j := startLine; j <= endLine; j++ {
				contextLines = append(contextLines, strings.TrimSpace(lines[j]))
				lineNums = append(lineNums, j+1)
			}

			snippet := extractSnippet(line, terms.Include)
			matches = append(matches, SearchMatch{
				Line:    i + 1,
				Snippet: snippet,
				Lines:   lineNums,
				Context: contextLines,
			})

			// 下一次匹配跳过后两行
			skipUntil = i + 3

			if len(matches) >= limit {
				break
			}
		}
	}

	return matches
}

// extractSnippet 提取匹配片段，高亮关键词
func extractSnippet(line string, includeTerms []string) string {
	// Limit snippet length
	maxLen := 200
	if len(line) <= maxLen {
		return strings.TrimSpace(line)
	}

	// Find best position to cut (near match term if possible)
	bestPos := 0
	lowerLine := strings.ToLower(line)

	for _, term := range includeTerms {
		pos := strings.Index(lowerLine, strings.ToLower(term))
		if pos > 0 {
			// Start a bit before the match
			start := pos - 30
			if start < 0 {
				start = 0
			}
			// Adjust to not cut in middle of word
			if start > 0 {
				for start > 0 && lowerLine[start-1] != ' ' {
					start--
				}
			}
			bestPos = start
			break
		}
	}

	// Cut and add ellipsis
	snippet := line[bestPos:min(bestPos+maxLen, len(line))]
	if bestPos > 0 {
		snippet = "..." + strings.TrimLeft(snippet, " ")
	}
	if bestPos+maxLen < len(line) {
		snippet = strings.TrimRight(snippet, " ") + "..."
	}

	return strings.TrimSpace(snippet)
}

// countMatches 统计总匹配数
func countMatches(results []SearchResult) int {
	total := 0
	for _, r := range results {
		total += len(r.Matches)
	}
	return total
}
