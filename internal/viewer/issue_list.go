package viewer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// issueListData 从 session 的 tool calls 中提取的审计问题清单
type issueListData struct {
	ID          string         `json:"id"`
	EncodedRepo string         `json:"encodedRepo"`
	RepoName    string         `json:"repoName"`
	SessionID   string         `json:"sessionID"`
	SessionTime time.Time      `json:"sessionTime"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Model       string         `json:"model"`
	Issues      []reviewIssueItem `json:"issues"`
	FileCount   int            `json:"fileCount"`
}

type reviewIssueItem struct {
	Path         string `json:"path"`
	Content      string `json:"content"`
	Severity     string `json:"severity"`
	Category     string `json:"category"`
	Suggestion   string `json:"suggestion,omitempty"`
	ExistingCode string `json:"existingCode,omitempty"`
	LineStart    int    `json:"lineStart,omitempty"`
	LineEnd      int    `json:"lineEnd,omitempty"`
}

func issueListDir(root string) string {
	return filepath.Join(root, "_issue_lists")
}

func issueListPath(root, issueListID string) string {
	return filepath.Join(issueListDir(root), issueListID+".json")
}

func handleGenerateIssueListAPI(w http.ResponseWriter, r *http.Request, root, encodedRepo, sessionID string) {
	vs, err := LoadSession(root, encodedRepo, sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("failed to load session: %v", err)})
		return
	}
	if vs.Summary.Status != "completed" && vs.Summary.Status != "completed_with_warnings" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "only completed sessions can generate issue lists"})
		return
	}

	issues := extractReviewIssueItems(vs)
	if len(issues) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no issues found in this session"})
		return
	}

	repoName := filepath.Base(vs.Summary.CWD)
	if repoName == "." || repoName == "" {
		repoName = encodedRepo
	}

	// 统计涉及的文件数
	uniqueFiles := make(map[string]struct{})
	for _, iss := range issues {
		uniqueFiles[iss.Path] = struct{}{}
	}

	listData := issueListData{
		ID:          sessionID,
		EncodedRepo: encodedRepo,
		RepoName:    repoName,
		SessionID:   sessionID,
		SessionTime: vs.Summary.Timestamp,
		GeneratedAt: time.Now(),
		Model:       vs.Summary.Model,
		Issues:      issues,
		FileCount:   len(uniqueFiles),
	}

	if err := writeIssueList(root, listData); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, listData)
}

func handleListIssueListsAPI(w http.ResponseWriter, r *http.Request, root string) {
	lists, err := listIssueLists(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	repoFilter := strings.TrimSpace(r.URL.Query().Get("repo"))
	filtered := make([]issueListData, 0, len(lists))
	for _, list := range lists {
		if repoFilter != "" && list.EncodedRepo != repoFilter {
			continue
		}
		filtered = append(filtered, list)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"issueLists": filtered})
}

func handleGetIssueListAPI(w http.ResponseWriter, _ *http.Request, root, repo, issueListID string) {
	listData, err := readIssueList(root, issueListID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if repo != "" && listData.EncodedRepo != repo {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issue list not found for this repo"})
		return
	}
	writeJSON(w, http.StatusOK, listData)
}

func extractReviewIssueItems(vs *ViewSession) []reviewIssueItem {
	issues := make([]reviewIssueItem, 0)
	seen := make(map[string]bool)

	for _, file := range vs.Files {
		for _, cards := range file.Tasks {
			for _, card := range cards {
				for _, tc := range card.ToolCalls {
					if tc.Name != "code_comment" || tc.Arguments == "" {
						continue
					}
					var payload struct {
						Path     string          `json:"path"`
						Comments json.RawMessage `json:"comments"`
					}
					if err := json.Unmarshal([]byte(tc.Arguments), &payload); err != nil {
						continue
					}

					filePath := payload.Path
					if filePath == "" {
						filePath = file.FilePath
					}

					var comments []map[string]interface{}
					if err := json.Unmarshal(payload.Comments, &comments); err != nil {
						continue
					}

					for _, c := range comments {
						content, _ := c["content"].(string)
						if content == "" {
							continue
						}

						suggestion, _ := c["suggestion_code"].(string)
						existingCode, _ := c["existing_code"].(string)
						lineStart, _ := c["line_start"].(float64)
						lineEnd, _ := c["line_end"].(float64)

						// 推断严重度
						severity := inferSeverity(content)
						// 推断分类
						category := inferCategory(content)

						dedupKey := fmt.Sprintf("%s:%s:%d", filePath, content[:minInt(len(content), 80)], int(lineStart))
						if seen[dedupKey] {
							continue
						}
						seen[dedupKey] = true

						issues = append(issues, reviewIssueItem{
							Path:         filePath,
							Content:      content,
							Severity:     severity,
							Category:     category,
							Suggestion:   suggestion,
							ExistingCode: existingCode,
							LineStart:    int(lineStart),
							LineEnd:      int(lineEnd),
						})
					}
				}
			}
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		order := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		getOrder := func(s string) int {
			if o, ok := order[strings.ToLower(s)]; ok {
				return o
			}
			return 99
		}
		oi, oj := getOrder(issues[i].Severity), getOrder(issues[j].Severity)
		if oi != oj {
			return oi < oj
		}
		return issues[i].Path < issues[j].Path
	})
	return issues
}

func inferSeverity(content string) string {
	lower := strings.ToLower(content)
	criticalKeywords := []string{"security", "sql injection", "xss", "traversal", "injection", "command injection", "unauthorized", "authentication bypass", "data leak", "credential", "secret exposed"}
	highKeywords := []string{"race condition", "deadlock", "memory leak", "panic", "crash", "nil pointer", "index out of range", "overflow", "buffer", "concurrency bug"}
	mediumKeywords := []string{"error handling", "missing validation", "hardcoded", "deprecated", "inefficient", "performance", "resource leak", "not thread-safe"}

	for _, kw := range criticalKeywords {
		if strings.Contains(lower, kw) {
			return "critical"
		}
	}
	for _, kw := range highKeywords {
		if strings.Contains(lower, kw) {
			return "high"
		}
	}
	for _, kw := range mediumKeywords {
		if strings.Contains(lower, kw) {
			return "medium"
		}
	}
	return "low"
}

func inferCategory(content string) string {
	lower := strings.ToLower(content)
	categories := []struct {
		keyword string
		cat     string
	}{
		{"security", "安全"},
		{"performance", "性能"},
		{"error handling", "错误处理"},
		{"concurrency", "并发"},
		{"memory", "内存"},
		{"validation", "输入验证"},
		{"hardcoded", "硬编码"},
		{"test", "测试"},
		{"style", "代码风格"},
		{"maintainability", "可维护性"},
	}
	for _, c := range categories {
		if strings.Contains(lower, c.keyword) {
			return c.cat
		}
	}
	return "其他"
}

func listIssueLists(root string) ([]issueListData, error) {
	entries, err := os.ReadDir(issueListDir(root))
	if err != nil {
		if os.IsNotExist(err) {
			return []issueListData{}, nil
		}
		return nil, err
	}
	lists := make([]issueListData, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		listData, err := readIssueList(root, strings.TrimSuffix(entry.Name(), ".json"))
		if err == nil {
			lists = append(lists, listData)
		}
	}
	sort.Slice(lists, func(i, j int) bool {
		return lists[i].GeneratedAt.After(lists[j].GeneratedAt)
	})
	return lists, nil
}

func readIssueList(root, issueListID string) (issueListData, error) {
	data, err := os.ReadFile(issueListPath(root, issueListID))
	if err != nil {
		return issueListData{}, err
	}
	var listData issueListData
	if err := json.Unmarshal(data, &listData); err != nil {
		return issueListData{}, err
	}
	return listData, nil
}

func writeIssueList(root string, listData issueListData) error {
	dir := issueListDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := issueListPath(root, listData.ID)
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	data, err := json.MarshalIndent(listData, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
