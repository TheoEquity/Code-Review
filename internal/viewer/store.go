// Package viewer provides a read-only WebUI for browsing session records
// produced by open-code-review runs. It scans JSONL files under
// $HOME/.opencodereview/sessions/, parses them, and exposes structured data.
package viewer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionsRoot returns the root directory where session JSONL files are stored.
func SessionsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".opencodereview", "sessions"), nil
}

// RepoInfo represents a discovered repository from the sessions directory.
type RepoInfo struct {
	EncodedPath  string    `json:"encodedPath"` // encoded directory name on disk
	SessionCount int       `json:"sessionCount"`
	LastModified time.Time `json:"lastModified"`
}

// DiscoverRepos walks the sessions root and returns one entry per subdirectory.
func DiscoverRepos(root string) ([]RepoInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var repos []RepoInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		repoDir := filepath.Join(root, e.Name())
		info := RepoInfo{EncodedPath: e.Name()}

		subEntries, err := os.ReadDir(repoDir)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if strings.HasSuffix(se.Name(), ".jsonl") {
				info.SessionCount++
				if fi, err := se.Info(); err == nil {
					if fi.ModTime().After(info.LastModified) {
						info.LastModified = fi.ModTime()
					}
				}
			}
		}
		if info.SessionCount > 0 {
			repos = append(repos, info)
		}
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].LastModified.After(repos[j].LastModified)
	})
	return repos, nil
}

// SessionSummary is built from session_start and session_end records.
type SessionSummary struct {
	SessionID     string    `json:"sessionID"`
	EncodedRepo   string    `json:"encodedRepo,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	CWD           string    `json:"cwd"`
	RepoName      string    `json:"repoName,omitempty"`
	RepoPath      string    `json:"repoPath,omitempty"`
	GitBranch     string    `json:"gitBranch"`
	Model         string    `json:"model"`
	ReviewMode    string    `json:"reviewMode"`
	DiffFrom      string    `json:"diffFrom"`
	DiffTo        string    `json:"diffTo"`
	DiffCommit    string    `json:"diffCommit"`
	FilesReviewed []string  `json:"filesReviewed"`
	DurationSec   float64   `json:"durationSec"`
	FileCount     int       `json:"fileCount"`
	LLMFailures   int       `json:"llmFailures"`
	WarningCount   int       `json:"warningCount,omitempty"`
	Status        string    `json:"status"`
}

// ListSessions returns lightweight summaries for all sessions in a repo subdir.
func ListSessions(root, encodedRepo string) ([]SessionSummary, error) {
	repoDir := filepath.Join(root, encodedRepo)
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return nil, fmt.Errorf("read repo dir: %w", err)
	}

	var summaries []SessionSummary
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		sessionID := strings.TrimSuffix(e.Name(), ".jsonl")
		s, err := peekSession(filepath.Join(repoDir, e.Name()))
		if err != nil {
			continue // skip unreadable files
		}
		if s.Timestamp.IsZero() || s.CWD == "" {
			continue
		}
		s.SessionID = sessionID
		s.EncodedRepo = encodedRepo
		summaries = append(summaries, s)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Timestamp.After(summaries[j].Timestamp)
	})
	return summaries, nil
}

// peekSession reads only the first and last record of a JSONL file.
func peekSession(path string) (SessionSummary, error) {
	f, err := os.Open(path)
	if err != nil {
		return SessionSummary{}, err
	}
	defer f.Close()

	var summary SessionSummary
	seenFiles := make(map[string]struct{})
	hasSessionEnd := false
	mainFailures := 0
	warningFailures := 0
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var lastLine []byte
	for scanner.Scan() {
		line := scanner.Bytes()
		lastLine = append([]byte(nil), line...)

		if summary.Timestamp.IsZero() {
			var rec map[string]any
			if err := json.Unmarshal(line, &rec); err != nil {
				continue
			}
			if ts, ok := rec["timestamp"].(string); ok {
				summary.Timestamp, _ = time.Parse(time.RFC3339, ts)
			}
			if cwd, ok := rec["cwd"].(string); ok {
				summary.CWD = cwd
			}
			if branch, ok := rec["gitBranch"].(string); ok {
				summary.GitBranch = branch
			}
			if model, ok := rec["model"].(string); ok {
				summary.Model = model
			}
			if rm, ok := rec["reviewMode"].(string); ok {
				summary.ReviewMode = rm
			}
			if v, ok := rec["diffFrom"].(string); ok {
				summary.DiffFrom = v
			}
			if v, ok := rec["diffTo"].(string); ok {
				summary.DiffTo = v
			}
			if v, ok := rec["diffCommit"].(string); ok {
				summary.DiffCommit = v
			}
		} else {
			var rec map[string]any
			if err := json.Unmarshal(line, &rec); err == nil {
				if filePath, ok := rec["filePath"].(string); ok && filePath != "" {
					seenFiles[filePath] = struct{}{}
				}
				if typ, _ := rec["type"].(string); typ == "llm_error" {
					if taskType, _ := rec["taskType"].(string); taskType == string(ReLocationTask) {
						warningFailures++
					} else {
						mainFailures++
					}
				}
			}
		}
	}

	if len(lastLine) > 0 {
		var rec map[string]any
		if err := json.Unmarshal(lastLine, &rec); err == nil {
			if typ, _ := rec["type"].(string); typ == "session_end" {
				hasSessionEnd = true
				if dur, ok := rec["duration_seconds"].(float64); ok {
					summary.DurationSec = dur
				}
				if files, ok := rec["files_reviewed"].([]any); ok {
					summary.FilesReviewed = make([]string, 0, len(files))
					for _, fv := range files {
						if s, ok := fv.(string); ok {
							summary.FilesReviewed = append(summary.FilesReviewed, s)
						}
					}
				}
				if f, ok := rec["llm_failures"].(float64); ok {
					summary.LLMFailures = int(f)
				}
			}
		}
	}
	if summary.FilesReviewed == nil && len(seenFiles) > 0 {
		summary.FilesReviewed = make([]string, 0, len(seenFiles))
		for filePath := range seenFiles {
			summary.FilesReviewed = append(summary.FilesReviewed, filePath)
		}
		sort.Strings(summary.FilesReviewed)
	}
	summary.FileCount = len(summary.FilesReviewed)
	if hasSessionEnd {
		if summary.FileCount == 0 {
			summary.Status = "failed"
		} else if mainFailures > 0 {
			summary.Status = "failed"
		} else if warningFailures > 0 {
			summary.Status = "completed_with_warnings"
			summary.WarningCount = warningFailures
		} else {
			summary.Status = "completed"
		}
	} else if reviewProcessRunning(summary.CWD) {
		summary.Status = "running"
	} else {
		summary.Status = "failed"
	}
	return summary, scanner.Err()
}

// ViewSession holds fully parsed records for one session.
type ViewSession struct {
	Summary    SessionSummary    `json:"summary"`
	TokenUsage TokenUsageSummary `json:"tokenUsage"`
	Files      []*FileGroup      `json:"files"` // ordered by file path
}

// TokenUsageSummary aggregates token counts across the session.
type TokenUsageSummary struct {
	TotalPromptTokens     int              `json:"totalPromptTokens"`
	TotalCompletionTokens int              `json:"totalCompletionTokens"`
	TotalCacheReadTokens  int              `json:"totalCacheReadTokens"`
	TotalCacheWriteTokens int              `json:"totalCacheWriteTokens"`
	RequestCount          int              `json:"requestCount"`
	FileTokenBreakdown    []FileTokenUsage `json:"fileTokenBreakdown"`
}

// FileTokenUsage tracks token totals for a single file within a session.
type FileTokenUsage struct {
	FilePath         string `json:"filePath"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	CacheReadTokens  int    `json:"cacheReadTokens"`
	CacheWriteTokens int    `json:"cacheWriteTokens"`
}

// FileGroup aggregates records for a single file.
type FileGroup struct {
	FilePath string                   `json:"filePath"`
	Tasks    map[TaskType][]*TaskCard `json:"tasks"`
}

// TaskType mirrors session.TaskType.
type TaskType string

const (
	PlanTask              TaskType = "plan_task"
	MainTask              TaskType = "main_task"
	MemoryCompressionTask TaskType = "memory_compression_task"
	ReLocationTask        TaskType = "re_location_task"
)

// TaskCard links an LLM request with its response and tool calls.
type TaskCard struct {
	RequestMessages  any            `json:"requestMessages"` // preserved for display
	RequestNo        int            `json:"requestNo"`
	ResponseContent  string         `json:"responseContent"`
	ToolCalls        []ToolCallInfo `json:"toolCalls"`
	DurationMs       int64          `json:"durationMs"`
	Error            string         `json:"error"`
	Model            string         `json:"model"`
	PromptTokens     int            `json:"promptTokens"`
	CompletionTokens int            `json:"completionTokens"`
	CacheReadTokens  int            `json:"cacheReadTokens"`
	CacheWriteTokens int            `json:"cacheWriteTokens"`
}

// ToolCallInfo summarizes a single tool call.
type ToolCallInfo struct {
	Name       string `json:"name"`
	Arguments  string `json:"arguments"`
	Result     string `json:"result"`
	Ok         bool   `json:"ok"`
	DurationMs int64  `json:"durationMs"`
}

// LoadSession fully parses a JSONL file into a ViewSession.
func LoadSession(root, encodedRepo, sessionID string) (*ViewSession, error) {
	path := filepath.Join(root, encodedRepo, sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session file: %w", err)
	}
	defer f.Close()

	vs := &ViewSession{Files: make([]*FileGroup, 0)}
	fileIndex := make(map[string]*FileGroup)
	hasSessionEnd := false
	mainFailures := 0
	warningFailures := 0

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		var rec map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue // skip malformed lines
		}
		typ, _ := rec["type"].(string)

		switch typ {
		case "session_start":
			if ts, ok := rec["timestamp"].(string); ok {
				vs.Summary.Timestamp, _ = time.Parse(time.RFC3339, ts)
			}
			if cwd, ok := rec["cwd"].(string); ok {
				vs.Summary.CWD = cwd
			}
			if branch, ok := rec["gitBranch"].(string); ok {
				vs.Summary.GitBranch = branch
			}
			if model, ok := rec["model"].(string); ok {
				vs.Summary.Model = model
			}
			if rm, ok := rec["reviewMode"].(string); ok {
				vs.Summary.ReviewMode = rm
			}
			if v, ok := rec["diffFrom"].(string); ok {
				vs.Summary.DiffFrom = v
			}
			if v, ok := rec["diffTo"].(string); ok {
				vs.Summary.DiffTo = v
			}
			if v, ok := rec["diffCommit"].(string); ok {
				vs.Summary.DiffCommit = v
			}

		case "llm_request":
			fp, _ := rec["filePath"].(string)
			tt, _ := rec["taskType"].(string)
			reqNo := 0
			if n, ok := rec["request_no"].(float64); ok {
				reqNo = int(n)
			}
			msgs := rec["messages"]

			tc := &TaskCard{RequestMessages: msgs, RequestNo: reqNo}

			fg := fileIndex[fp]
			if fg == nil {
				fg = &FileGroup{FilePath: fp, Tasks: make(map[TaskType][]*TaskCard)}
				fileIndex[fp] = fg
				vs.Files = append(vs.Files, fg)
			}
			fg.Tasks[TaskType(tt)] = append(fg.Tasks[TaskType(tt)], tc)

		case "llm_response":
			fp, _ := rec["filePath"].(string)
			content, _ := rec["content"].(string)
			durationMs := int64(0)
			if d, ok := rec["duration_ms"].(float64); ok {
				durationMs = int64(d)
			}
			model, _ := rec["model"].(string)
			errStr, _ := rec["error"].(string)

			promptTok := 0
			completionTok := 0
			cacheReadTok := 0
			cacheWriteTok := 0
			if usage, ok := rec["usage"].(map[string]any); ok {
				if v, ok := usage["prompt_tokens"].(float64); ok {
					promptTok = int(v)
				}
				if v, ok := usage["completion_tokens"].(float64); ok {
					completionTok = int(v)
				}
				if v, ok := usage["cache_read_tokens"].(float64); ok {
					cacheReadTok = int(v)
				}
				if v, ok := usage["cache_write_tokens"].(float64); ok {
					cacheWriteTok = int(v)
				}
			}

			tt, _ := rec["taskType"].(string)
			fg := fileIndex[fp]
			if fg != nil {
				cards := fg.Tasks[TaskType(tt)]
				if len(cards) > 0 && cards[len(cards)-1].ResponseContent == "" {
					card := cards[len(cards)-1]
					card.ResponseContent = content
					card.DurationMs = durationMs
					card.Model = model
					card.Error = errStr
					card.PromptTokens = promptTok
					card.CompletionTokens = completionTok
					card.CacheReadTokens = cacheReadTok
					card.CacheWriteTokens = cacheWriteTok
				}
			}

			// Also attach tool_calls to the same card
			if tcs, ok := rec["tool_calls"].([]any); ok && fg != nil {
				tt, _ := rec["taskType"].(string)
				cards := fg.Tasks[TaskType(tt)]
				if len(cards) > 0 {
					card := cards[len(cards)-1]
					for _, tc := range tcs {
						if tm, ok := tc.(map[string]any); ok {
							name, _ := tm["name"].(string)
							args, _ := tm["arguments"].(string)
							info := ToolCallInfo{Name: name, Arguments: args}
							if name == "task_done" {
								info.Ok = true
							}
							card.ToolCalls = append(card.ToolCalls, info)
						}
					}
				}
			}

		case "llm_error":
			fp, _ := rec["filePath"].(string)
			tt, _ := rec["taskType"].(string)
			errStr, _ := rec["error"].(string)
			if tt == string(ReLocationTask) {
				warningFailures++
			} else {
				mainFailures++
			}
			durationMs := int64(0)
			if d, ok := rec["duration_ms"].(float64); ok {
				durationMs = int64(d)
			}

			fg := fileIndex[fp]
			if fg != nil {
				cards := fg.Tasks[TaskType(tt)]
				if len(cards) > 0 && cards[len(cards)-1].Error == "" {
					card := cards[len(cards)-1]
					card.Error = errStr
					card.DurationMs = durationMs
				}
			}

		case "tool_call":
			result, _ := rec["result"].(string)
			okVal := true
			if b, hasOk := rec["ok"].(bool); hasOk {
				okVal = b
			}
			fp, _ := rec["filePath"].(string)
			tt, _ := rec["taskType"].(string)
			durationMs := int64(0)
			if d, ok2 := rec["duration_ms"].(float64); ok2 {
				durationMs = int64(d)
			}

			fg := fileIndex[fp]
			if fg != nil {
				cards := fg.Tasks[TaskType(tt)]
				if len(cards) > 0 {
					card := cards[len(cards)-1]
					for ti := range card.ToolCalls {
						if card.ToolCalls[ti].Result == "" && !card.ToolCalls[ti].Ok {
							card.ToolCalls[ti].Result = result
							card.ToolCalls[ti].Ok = okVal
							card.ToolCalls[ti].DurationMs = durationMs
							break
						}
					}
				}
			}

		case "session_end":
			hasSessionEnd = true
			if dur, ok := rec["duration_seconds"].(float64); ok {
				vs.Summary.DurationSec = dur
			}
			if files, ok := rec["files_reviewed"].([]any); ok {
				vs.Summary.FilesReviewed = make([]string, 0, len(files))
				for _, fv := range files {
					if s, ok2 := fv.(string); ok2 {
						vs.Summary.FilesReviewed = append(vs.Summary.FilesReviewed, s)
					}
				}
			}
			vs.Summary.FileCount = len(vs.Summary.FilesReviewed)
			if f, ok := rec["llm_failures"].(float64); ok {
				vs.Summary.LLMFailures = int(f)
			}
		}
	}
	if hasSessionEnd {
		if vs.Summary.FileCount == 0 {
			vs.Summary.Status = "failed"
		} else if mainFailures > 0 {
			vs.Summary.Status = "failed"
		} else if warningFailures > 0 {
			vs.Summary.Status = "completed_with_warnings"
			vs.Summary.WarningCount = warningFailures
		} else {
			vs.Summary.Status = "completed"
		}
	} else if reviewProcessRunning(vs.Summary.CWD) {
		vs.Summary.Status = "running"
	} else {
		vs.Summary.Status = "failed"
	}

	// Aggregate token usage across all task cards
	fileBreakdown := make([]FileTokenUsage, 0, len(vs.Files))
	for _, fg := range vs.Files {
		ft := FileTokenUsage{FilePath: fg.FilePath}
		for _, cards := range fg.Tasks {
			for _, c := range cards {
				vs.TokenUsage.TotalPromptTokens += c.PromptTokens
				vs.TokenUsage.TotalCompletionTokens += c.CompletionTokens
				vs.TokenUsage.TotalCacheReadTokens += c.CacheReadTokens
				vs.TokenUsage.TotalCacheWriteTokens += c.CacheWriteTokens
				if c.ResponseContent != "" || c.PromptTokens > 0 {
					vs.TokenUsage.RequestCount++
				}
				ft.PromptTokens += c.PromptTokens
				ft.CompletionTokens += c.CompletionTokens
				ft.CacheReadTokens += c.CacheReadTokens
				ft.CacheWriteTokens += c.CacheWriteTokens
			}
		}
		fileBreakdown = append(fileBreakdown, ft)
	}
	sort.Slice(fileBreakdown, func(i, j int) bool {
		return fileBreakdown[i].PromptTokens+fileBreakdown[i].CompletionTokens > fileBreakdown[j].PromptTokens+fileBreakdown[j].CompletionTokens
	})
	vs.TokenUsage.FileTokenBreakdown = fileBreakdown

	sort.Slice(vs.Files, func(i, j int) bool {
		return vs.Files[i].FilePath < vs.Files[j].FilePath
	})

	vs.Summary.SessionID = sessionID
	return vs, scanner.Err()
}

func reviewProcessRunning(repoPath string) bool {
	if repoPath == "" {
		return false
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || !isDigits(entry.Name()) {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		if len(args) < 4 {
			continue
		}
		for i, arg := range args {
			if arg == "--repo" && i+1 < len(args) && args[i+1] == repoPath && containsArg(args, "review") {
				return true
			}
		}
	}
	return false
}

func containsArg(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
