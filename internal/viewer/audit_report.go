package viewer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/open-code-review/open-code-review/internal/llm"
)

type auditReport struct {
	ID           string                 `json:"id"`
	EncodedRepo  string                 `json:"encodedRepo"`
	RepoName     string                 `json:"repoName"`
	SessionID    string                 `json:"sessionID"`
	SessionTime  time.Time              `json:"sessionTime"`
	GeneratedAt  time.Time              `json:"generatedAt"`
	Model        string                 `json:"model"`
	IssueCount   int                    `json:"issueCount"`
	FileCount    int                    `json:"fileCount"`
	Status       string                 `json:"status"`
	Content      string                 `json:"content"`
	Structured   *auditReportStructured `json:"structured,omitempty"`
	PromptTokens int64                  `json:"promptTokens,omitempty"`
	OutputTokens int64                  `json:"outputTokens,omitempty"`
}

type auditReportStructured struct {
	Title            string                   `json:"title"`
	ExecutiveSummary string                   `json:"executiveSummary"`
	RiskLevel        string                   `json:"riskLevel"`
	RiskReason       string                   `json:"riskReason"`
	Categories       []auditReportCategory    `json:"categories"`
	KeyModules       []auditReportKeyModule   `json:"keyModules"`
	TopFixes         []auditReportTopFix      `json:"topFixes"`
	Roadmap          []auditReportRoadmapItem `json:"roadmap"`
	ManualReview     []string                 `json:"manualReview"`
}

type auditReportCategory struct {
	Name        string `json:"name"`
	RiskLevel   string `json:"riskLevel"`
	Count       int    `json:"count,omitempty"`
	Description string `json:"description"`
}

type auditReportKeyModule struct {
	Path       string `json:"path"`
	RiskLevel  string `json:"riskLevel"`
	Finding    string `json:"finding"`
	Suggestion string `json:"suggestion"`
}

type auditReportTopFix struct {
	Priority   int    `json:"priority"`
	RiskLevel  string `json:"riskLevel"`
	Title      string `json:"title"`
	File       string `json:"file"`
	Impact     string `json:"impact"`
	Suggestion string `json:"suggestion"`
}

type auditReportRoadmapItem struct {
	Phase string   `json:"phase"`
	Goal  string   `json:"goal"`
	Items []string `json:"items"`
}

type auditReportIssue struct {
	Path           string `json:"path"`
	Content        string `json:"content"`
	SuggestionCode string `json:"suggestionCode,omitempty"`
	ExistingCode   string `json:"existingCode,omitempty"`
	Thinking       string `json:"thinking,omitempty"`
}

type auditReportListResponse struct {
	Reports []auditReport `json:"reports"`
}

func auditReportDir(root string) string {
	return filepath.Join(root, "_audit_reports")
}

func auditReportPath(root, reportID string) string {
	return filepath.Join(auditReportDir(root), reportID+".json")
}

func handleListAuditReportsAPI(w http.ResponseWriter, r *http.Request, root string) {
	reports, err := listAuditReports(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	repoFilter := strings.TrimSpace(r.URL.Query().Get("repo"))
	dateFilter := strings.TrimSpace(r.URL.Query().Get("date"))
	filtered := make([]auditReport, 0, len(reports))
	for _, report := range reports {
		if repoFilter != "" && report.EncodedRepo != repoFilter {
			continue
		}
		if dateFilter != "" && report.SessionTime.Format("2006-01-02") != dateFilter {
			continue
		}
		filtered = append(filtered, report)
	}
	writeJSON(w, http.StatusOK, auditReportListResponse{Reports: filtered})
}

func handleGetAuditReportAPI(w http.ResponseWriter, _ *http.Request, root, reportID string) {
	report, err := readAuditReport(root, reportID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func handleGenerateAuditReportAPI(w http.ResponseWriter, _ *http.Request, root, encodedRepo, sessionID string) {
	vs, err := LoadSession(root, encodedRepo, sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("failed to load session: %v", err)})
		return
	}
	if vs.Summary.Status != "completed" && vs.Summary.Status != "completed_with_warnings" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "only completed sessions can generate audit reports"})
		return
	}

	issues := extractAuditReportIssues(vs)
	if len(issues) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no issues found in this session"})
		return
	}

	cfgPath, err := viewerConfigPath()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	ep, err := llm.ResolveEndpoint(cfgPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	prompt := buildAuditReportPrompt(vs, issues)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	client := llm.NewLLMClient(ep)
	resp, content, err := generateCompleteAuditReport(ctx, client, ep.Model, prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("generate audit report: %v", err)})
		return
	}

	repoName := filepath.Base(vs.Summary.CWD)
	if repoName == "." || repoName == "" {
		repoName = encodedRepo
	}
	structured := parseStructuredAuditReport(content)
	report := auditReport{
		ID:          sessionID,
		EncodedRepo: encodedRepo,
		RepoName:    repoName,
		SessionID:   sessionID,
		SessionTime: vs.Summary.Timestamp,
		GeneratedAt: time.Now(),
		Model:       ep.Model,
		IssueCount:  len(issues),
		FileCount:   vs.Summary.FileCount,
		Status:      vs.Summary.Status,
		Content:     content,
		Structured:  structured,
	}
	if resp.Usage != nil {
		report.PromptTokens = resp.Usage.PromptTokens
		report.OutputTokens = resp.Usage.CompletionTokens
	}
	if err := writeAuditReport(root, report); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, report)
}

func generateCompleteAuditReport(ctx context.Context, client llm.LLMClient, model, prompt string) (*llm.ChatResponse, string, error) {
	baseMessages := []llm.Message{
		llm.NewTextMessage("system", "You are a senior software audit lead. Produce concise, actionable Chinese audit reports."),
		llm.NewTextMessage("user", prompt),
	}
	resp, err := client.CompletionsWithCtx(ctx, llm.ChatRequest{
		Model:     model,
		Messages:  baseMessages,
		MaxTokens: 8192,
	})
	if err != nil {
		return nil, "", err
	}
	content := resp.Content()
	lastResp := resp
	for i := 0; i < 2 && needsAuditReportContinuation(content); i++ {
		contResp, err := client.CompletionsWithCtx(ctx, llm.ChatRequest{
			Model: model,
			Messages: append(baseMessages,
				llm.NewTextMessage("assistant", content),
				llm.NewTextMessage("user", "报告在上一条回复中被截断。请从截断处继续，只输出剩余内容，继续完成第5项后续内容、第6项和第7项，不要重复已经输出的内容。"),
			),
			MaxTokens: 4096,
		})
		if err != nil {
			return lastResp, content, nil
		}
		part := strings.TrimSpace(contResp.Content())
		if part == "" {
			break
		}
		content = strings.TrimSpace(content) + "\n\n" + part
		lastResp = contResp
	}
	return lastResp, content, nil
}

func needsAuditReportContinuation(content string) bool {
	normalized := strings.ReplaceAll(content, " ", "")
	return !strings.Contains(normalized, "##6.") &&
		!strings.Contains(normalized, "##6、") &&
		!strings.Contains(normalized, "##6") &&
		!strings.Contains(normalized, "##7.") &&
		!strings.Contains(normalized, "需要人工复核")
}

func listAuditReports(root string) ([]auditReport, error) {
	entries, err := os.ReadDir(auditReportDir(root))
	if err != nil {
		if os.IsNotExist(err) {
			return []auditReport{}, nil
		}
		return nil, err
	}
	reports := make([]auditReport, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		report, err := readAuditReport(root, strings.TrimSuffix(entry.Name(), ".json"))
		if err == nil {
			reports = append(reports, report)
		}
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].GeneratedAt.After(reports[j].GeneratedAt)
	})
	return reports, nil
}

func readAuditReport(root, reportID string) (auditReport, error) {
	data, err := os.ReadFile(auditReportPath(root, reportID))
	if err != nil {
		return auditReport{}, err
	}
	var report auditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return auditReport{}, err
	}
	return report, nil
}

func writeAuditReport(root string, report auditReport) error {
	dir := auditReportDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(auditReportPath(root, report.ID), data, 0o644)
}

func parseStructuredAuditReport(content string) *auditReportStructured {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	trimmed = strings.TrimSpace(trimmed)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return nil
	}
	var report auditReportStructured
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &report); err != nil {
		return nil
	}
	if report.Title == "" && report.ExecutiveSummary == "" {
		return nil
	}
	return &report
}

func extractAuditReportIssues(vs *ViewSession) []auditReportIssue {
	seen := make(map[string]struct{})
	issues := make([]auditReportIssue, 0)
	for _, file := range vs.Files {
		for _, cards := range file.Tasks {
			for _, card := range cards {
				for _, call := range card.ToolCalls {
					if call.Name != "code_comment" || call.Arguments == "" {
						continue
					}
					for _, issue := range parseAuditIssuesFromToolCall(file.FilePath, call.Arguments) {
						key := issue.Path + "\n" + issue.Content + "\n" + issue.SuggestionCode
						if _, ok := seen[key]; ok {
							continue
						}
						seen[key] = struct{}{}
						issues = append(issues, issue)
					}
				}
			}
		}
	}
	return issues
}

func parseAuditIssuesFromToolCall(defaultPath, rawArgs string) []auditReportIssue {
	var payload map[string]any
	if err := json.Unmarshal([]byte(rawArgs), &payload); err != nil {
		return nil
	}
	path, _ := payload["path"].(string)
	if path == "" {
		path = defaultPath
	}
	rawComments := payload["comments"]
	if commentString, ok := rawComments.(string); ok {
		var parsed []any
		if err := json.Unmarshal([]byte(commentString), &parsed); err == nil {
			rawComments = parsed
		}
	}
	comments, ok := rawComments.([]any)
	if !ok {
		return nil
	}
	issues := make([]auditReportIssue, 0, len(comments))
	for _, raw := range comments {
		obj, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		content, _ := obj["content"].(string)
		if strings.TrimSpace(content) == "" || path == "" {
			continue
		}
		issue := auditReportIssue{Path: path, Content: strings.TrimSpace(content)}
		issue.SuggestionCode, _ = obj["suggestion_code"].(string)
		issue.ExistingCode, _ = obj["existing_code"].(string)
		issue.Thinking, _ = obj["thinking"].(string)
		issues = append(issues, issue)
	}
	return issues
}

func buildAuditReportPrompt(vs *ViewSession, issues []auditReportIssue) string {
	issueLimit := 120
	if len(issues) < issueLimit {
		issueLimit = len(issues)
	}
	var sb strings.Builder
	sb.WriteString("请基于以下代码审计问题清单生成结构化综合诊断报告。\n")
	sb.WriteString("必须只输出合法 JSON，不要输出 Markdown，不要使用代码块。JSON schema 如下：\n")
	sb.WriteString(`{
  "title": "string",
  "executiveSummary": "string",
  "riskLevel": "Critical|High|Medium|Low",
  "riskReason": "string",
  "categories": [{"name":"string","riskLevel":"Critical|High|Medium|Low","count":0,"description":"string"}],
  "keyModules": [{"path":"string","riskLevel":"Critical|High|Medium|Low","finding":"string","suggestion":"string"}],
  "topFixes": [{"priority":1,"riskLevel":"Critical|High|Medium|Low","title":"string","file":"string","impact":"string","suggestion":"string"}],
  "roadmap": [{"phase":"string","goal":"string","items":["string"]}],
  "manualReview": ["string"]
}` + "\n\n")
	sb.WriteString("要求：executiveSummary 2-4 句话；categories 4-8 项；keyModules 5-10 项；topFixes 10 项以内；roadmap 3-5 个阶段；manualReview 5-10 项。\n\n")
	sb.WriteString(fmt.Sprintf("仓库路径：%s\n", vs.Summary.CWD))
	sb.WriteString(fmt.Sprintf("审计模式：%s\n", vs.Summary.ReviewMode))
	sb.WriteString(fmt.Sprintf("已审查文件数：%d\n", vs.Summary.FileCount))
	sb.WriteString(fmt.Sprintf("问题总数：%d\n", len(issues)))
	sb.WriteString(fmt.Sprintf("以下列出前 %d 条去重问题；如果问题数量更多，请基于样本和分布给出总体判断。\n\n", issueLimit))
	for i := 0; i < issueLimit; i++ {
		issue := issues[i]
		sb.WriteString(fmt.Sprintf("## Issue %d\n", i+1))
		sb.WriteString(fmt.Sprintf("File: %s\n", issue.Path))
		sb.WriteString(fmt.Sprintf("Problem: %s\n", truncateForPrompt(issue.Content, 1200)))
		if issue.SuggestionCode != "" {
			sb.WriteString(fmt.Sprintf("Suggestion: %s\n", truncateForPrompt(issue.SuggestionCode, 800)))
		}
		if issue.Thinking != "" {
			sb.WriteString(fmt.Sprintf("Reasoning: %s\n", truncateForPrompt(issue.Thinking, 800)))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func truncateForPrompt(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
