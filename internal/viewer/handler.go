package viewer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	ruleconfig "github.com/open-code-review/open-code-review/internal/config/rules"
	"github.com/open-code-review/open-code-review/internal/gitcmd"
	"github.com/open-code-review/open-code-review/internal/llm"
)

type repoListResponse struct {
	Repos []repoListItem `json:"repos"`
}

type repoListItem struct {
	EncodedPath  string `json:"encodedPath"`
	DisplayName  string `json:"displayName"`
	RepoPath     string `json:"repoPath"`
	RemoteURL    string `json:"remoteURL,omitempty"`
	SessionCount int    `json:"sessionCount"`
	FileCount    int    `json:"fileCount,omitempty"`
	LastModified string `json:"lastModified"`
	HasToken     bool   `json:"hasToken,omitempty"`
}

type viewerRepoEntry struct {
	Name      string `json:"name"`
	LocalPath string `json:"localPath"`
	RemoteURL string `json:"remoteURL,omitempty"`
	Token     string `json:"token,omitempty"`
	FileCount int    `json:"fileCount,omitempty"`
}

type viewerRepoRegistry struct {
	Repos []viewerRepoEntry `json:"repos"`
}

type sessionListResponse struct {
	EncodedRepo string           `json:"encodedRepo"`
	RepoName    string           `json:"repoName"`
	Sessions    []SessionSummary `json:"sessions"`
}

type sessionDetailResponse struct {
	EncodedRepo string       `json:"encodedRepo"`
	RepoName    string       `json:"repoName"`
	Session     *ViewSession `json:"session"`
}

type rulesOverviewResponse struct {
	Layers []ruleLayerSummary `json:"layers"`
}

type ruleLayerSummary struct {
	Priority    int                           `json:"priority"`
	Source      string                        `json:"source"`
	Title       string                        `json:"title"`
	Path        string                        `json:"path"`
	Description string                        `json:"description,omitempty"`
	Available   bool                          `json:"available"`
	DefaultRule string                        `json:"defaultRule,omitempty"`
	Rules       []ruleconfig.ProjectRuleEntry `json:"rules,omitempty"`
	Include     []string                      `json:"include,omitempty"`
	Exclude     []string                      `json:"exclude,omitempty"`
	PathRules   []systemRuleEntry             `json:"pathRules,omitempty"`
}

type systemRuleEntry struct {
	Pattern string `json:"pattern"`
	Rule    string `json:"rule"`
}

type repoStatusResponse struct {
	EncodedRepo string           `json:"encodedRepo"`
	RepoName    string           `json:"repoName"`
	RepoPath    string           `json:"repoPath"`
	Branch      string           `json:"branch"`
	Head        string           `json:"head"`
	Stats       repoStatusStats  `json:"stats"`
	Files       []repoFileStatus `json:"files"`
}

type reviewTaskResponse struct {
	TaskID       string    `json:"taskID"`
	EncodedRepo  string    `json:"encodedRepo"`
	RepoName     string    `json:"repoName"`
	RepoPath     string    `json:"repoPath"`
	Branch       string    `json:"branch,omitempty"`
	ReviewMode   string    `json:"reviewMode,omitempty"`
	BaseRef      string    `json:"baseRef,omitempty"`
	TargetRef    string    `json:"targetRef,omitempty"`
	CommitRef    string    `json:"commitRef,omitempty"`
	Background   string    `json:"background,omitempty"`
	Format       string    `json:"format,omitempty"`
	Timeout      string    `json:"timeout,omitempty"`
	Concurrency  string    `json:"concurrency,omitempty"`
	RulePath     string    `json:"rulePath,omitempty"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt,omitempty"`
	ExitCode     int       `json:"exitCode,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Output       string    `json:"output,omitempty"`
}

type createReviewTaskRequest struct {
	EncodedRepo   string `json:"encodedRepo"`
	EncodedBranch string `json:"encodedBranch"`
	ReviewMode    string `json:"reviewMode"`
	BaseRef       string `json:"baseRef"`
	TargetRef     string `json:"targetRef"`
	CommitRef     string `json:"commitRef"`
	Background    string `json:"background"`
	Format        string `json:"format"`
	Timeout       string `json:"timeout"`
	Concurrency   string `json:"concurrency"`
	RulePath      string `json:"rulePath"`
}

type addRepoRequest struct {
	RepoName  string `json:"repoName"`
	LocalPath string `json:"localPath"`
	RemoteURL string `json:"remoteURL"`
	Token     string `json:"token"`
}

type llmConfigPayload struct {
	URL          string `json:"url"`
	AuthToken    string `json:"authToken"`
	Model        string `json:"model"`
	UseAnthropic bool   `json:"useAnthropic"`
	ExtraBody    string `json:"extraBody"`
}

type llmConfigResponse struct {
	Config      llmConfigPayload `json:"config"`
	ConfigPath  string           `json:"configPath"`
	Configured  bool             `json:"configured"`
	ResolvedURL string           `json:"resolvedUrl,omitempty"`
	ResolvedVia string           `json:"resolvedVia,omitempty"`
	Protocol    string           `json:"protocol,omitempty"`
}

type managedRepo struct {
	EncodedPath  string
	DisplayName  string
	RepoPath     string
	RemoteURL    string
	FileCount    int
	SessionCount int
	LastModified time.Time
}

var reviewTaskStore = struct {
	sync.Mutex
	tasks map[string]*reviewTaskResponse
}{tasks: make(map[string]*reviewTaskResponse)}

var viewerRepoStore = struct {
	sync.Mutex
}{}

type repoStatusStats struct {
	Staged    int `json:"staged"`
	Unstaged  int `json:"unstaged"`
	Untracked int `json:"untracked"`
	Conflicts int `json:"conflicts"`
	Total     int `json:"total"`
}

type repoFileStatus struct {
	Path           string `json:"path"`
	IndexStatus    string `json:"indexStatus"`
	WorktreeStatus string `json:"worktreeStatus"`
	Category       string `json:"category"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("[viewer] json encode error: %v\n", err)
	}
}

func viewerConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".opencodereview", "config.json"), nil
}

func viewerRepoRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".opencodereview", "viewer_repos.json"), nil
}

func managedRepoFromPath(root, repoPath string) managedRepo {
	item := managedRepo{
		EncodedPath: encodeRepoPath(repoPath),
		DisplayName: filepath.Base(repoPath),
		RepoPath:    repoPath,
	}
	summaries, err := ListSessions(root, item.EncodedPath)
	if err == nil {
		item.SessionCount = len(summaries)
		for _, summary := range summaries {
			if summary.Timestamp.After(item.LastModified) {
				item.LastModified = summary.Timestamp
			}
		}
	}
	return item
}

func readViewerRepoRegistry() (*viewerRepoRegistry, error) {
	path, err := viewerRepoRegistryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &viewerRepoRegistry{Repos: []viewerRepoEntry{}}, nil
		}
		return nil, fmt.Errorf("read viewer repo registry: %w", err)
	}
	var reg viewerRepoRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		var legacyRepos []string
		if legacyErr := json.Unmarshal(data, &legacyRepos); legacyErr != nil {
			return nil, fmt.Errorf("parse viewer repo registry: %w", err)
		}
		reg.Repos = make([]viewerRepoEntry, 0, len(legacyRepos))
		for _, repoPath := range legacyRepos {
			if strings.TrimSpace(repoPath) == "" || isLikelyGitURL(repoPath) || strings.HasPrefix(repoPath, urlRepoPrefix) {
				continue
			}
			reg.Repos = append(reg.Repos, viewerRepoEntry{
				Name:      filepath.Base(repoPath),
				LocalPath: repoPath,
			})
		}
	}
	if reg.Repos == nil {
		reg.Repos = []viewerRepoEntry{}
	}
	return &reg, nil
}

func writeViewerRepoRegistry(reg *viewerRepoRegistry) error {
	path, err := viewerRepoRegistryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create viewer repo dir: %w", err)
	}
	data, err := json.MarshalIndent(reg, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal viewer repo registry: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write viewer repo registry: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename viewer repo registry: %w", err)
	}
	return nil
}

func encodeViewerRepoName(name string) string {
	encoded := strings.TrimSpace(name)
	encoded = strings.NewReplacer("/", "_", " ", "_", "\\", "_").Replace(encoded)
	return encoded
}

func countRepoFiles(repoPath string) int {
	count := 0
	_ = filepath.Walk(repoPath, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func normalizeConfigString(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "\"'")
	return strings.TrimSpace(trimmed)
}

func readViewerConfig() (map[string]any, string, error) {
	configPath, err := viewerConfigPath()
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, configPath, nil
		}
		return nil, "", fmt.Errorf("read config: %w", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, "", fmt.Errorf("parse config: %w", err)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, configPath, nil
}

func readLLMConfigPayload() (llmConfigPayload, string, error) {
	cfg, configPath, err := readViewerConfig()
	if err != nil {
		return llmConfigPayload{}, "", err
	}
	payload := llmConfigPayload{UseAnthropic: true}
	if llmSection, ok := cfg["llm"].(map[string]any); ok {
		if value, ok := llmSection["url"].(string); ok {
			payload.URL = value
		}
		if value, ok := llmSection["auth_token"].(string); ok {
			payload.AuthToken = value
		}
		if value, ok := llmSection["model"].(string); ok {
			payload.Model = value
		}
		if value, ok := llmSection["use_anthropic"].(bool); ok {
			payload.UseAnthropic = value
		}
		if value, ok := llmSection["extra_body"]; ok && value != nil {
			if data, err := json.MarshalIndent(value, "", "  "); err == nil {
				payload.ExtraBody = string(data)
			}
		}
	}
	return payload, configPath, nil
}

func handleLLMConfigAPI(w http.ResponseWriter, _ *http.Request) {
	payload, configPath, err := readLLMConfigPayload()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	response := llmConfigResponse{
		Config:     payload,
		ConfigPath: configPath,
		Configured: payload.URL != "" && payload.AuthToken != "" && payload.Model != "",
	}
	if endpoint, err := llm.ResolveEndpoint(configPath); err == nil {
		response.ResolvedURL = endpoint.URL
		response.ResolvedVia = endpoint.Source
		response.Protocol = endpoint.Protocol
	}
	writeJSON(w, http.StatusOK, response)
}

func handleSaveLLMConfigAPI(w http.ResponseWriter, r *http.Request) {
	var req llmConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.URL = normalizeConfigString(req.URL)
	req.AuthToken = normalizeConfigString(req.AuthToken)
	req.Model = normalizeConfigString(req.Model)
	req.ExtraBody = strings.TrimSpace(req.ExtraBody)
	if req.URL == "" || req.AuthToken == "" || req.Model == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url, auth token, and model are required"})
		return
	}

	cfg, configPath, err := readViewerConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	llmSection := map[string]any{
		"url":           req.URL,
		"auth_token":    req.AuthToken,
		"model":         req.Model,
		"use_anthropic": req.UseAnthropic,
	}
	if req.ExtraBody != "" {
		var extraBody map[string]any
		if err := json.Unmarshal([]byte(req.ExtraBody), &extraBody); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid extraBody json: %v", err)})
			return
		}
		llmSection["extra_body"] = extraBody
	}
	cfg["llm"] = llmSection

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create config dir: %v", err)})
		return
	}
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("marshal config: %v", err)})
		return
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("write config: %v", err)})
		return
	}

	handleLLMConfigAPI(w, r)
}

func validatePathValue(value string) bool {
	return value != "" && !strings.Contains(value, "..") && !strings.Contains(value, "/")
}

func deriveRepoName(repo string, summaries []SessionSummary) string {
	name := repo
	for _, s := range summaries {
		if s.CWD != "" {
			name = filepath.Base(s.CWD)
			break
		}
	}
	return name
}

func encodeRepoPath(p string) string {
	if p == "" {
		return "empty"
	}
	vol := filepath.VolumeName(p)
	p = p[len(vol):]
	p = strings.TrimLeft(p, "/\\")
	p = strings.ReplaceAll(p, "/", "-")
	p = strings.ReplaceAll(p, "\\", "-")
	vol = strings.ReplaceAll(vol, ":", "_")
	result := vol + p
	if result == "" {
		return "empty"
	}
	return result
}

func currentRepoInfo(root string) (*managedRepo, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	runner := gitcmd.New(2)
	out, err := runner.Run(context.Background(), cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	repoPath := strings.TrimSpace(out)
	if repoPath == "" {
		return nil, fmt.Errorf("empty repo path")
	}
	item := managedRepoFromPath(root, repoPath)
	return &item, nil
}

func discoverManagedRepos(root string) ([]managedRepo, error) {
	repos, err := DiscoverRepos(root)
	if err != nil {
		return nil, err
	}
	items := make([]managedRepo, 0, len(repos)+1)
	seen := make(map[string]struct{})
	for _, repo := range repos {
		summaries, err := ListSessions(root, repo.EncodedPath)
		if err != nil || len(summaries) == 0 {
			items = append(items, managedRepo{
				EncodedPath:  repo.EncodedPath,
				DisplayName:  repo.EncodedPath,
				SessionCount: repo.SessionCount,
				LastModified: repo.LastModified,
			})
			seen[repo.EncodedPath] = struct{}{}
			continue
		}
		repoPath := summaries[0].CWD
		items = append(items, managedRepo{
			EncodedPath:  repo.EncodedPath,
			DisplayName:  deriveRepoName(repo.EncodedPath, summaries),
			RepoPath:     repoPath,
			SessionCount: repo.SessionCount,
			LastModified: repo.LastModified,
		})
		seen[repo.EncodedPath] = struct{}{}
	}
	registeredRepos, err := readViewerRepoRegistry()
	if err == nil {
		for _, entry := range registeredRepos.Repos {
			encoded := encodeRepoPath(entry.LocalPath)
			if _, ok := seen[encoded]; ok {
				for i := range items {
					if items[i].EncodedPath == encoded {
						items[i].DisplayName = entry.Name
						items[i].RepoPath = entry.LocalPath
						items[i].RemoteURL = entry.RemoteURL
						items[i].FileCount = entry.FileCount
						break
					}
				}
				continue
			}
			summaries, err := ListSessions(root, encoded)
			sessionCount := 0
			lastModified := time.Time{}
			if err == nil {
				sessionCount = len(summaries)
				for _, summary := range summaries {
					if summary.Timestamp.After(lastModified) {
						lastModified = summary.Timestamp
					}
				}
			}
			items = append(items, managedRepo{
				EncodedPath:  encoded,
				DisplayName:  entry.Name,
				RepoPath:     entry.LocalPath,
				RemoteURL:    entry.RemoteURL,
				FileCount:    entry.FileCount,
				SessionCount: sessionCount,
				LastModified: lastModified,
			})
			seen[encoded] = struct{}{}
		}
	}
	if currentRepo, err := currentRepoInfo(root); err == nil {
		if _, ok := seen[currentRepo.EncodedPath]; !ok {
			items = append(items, *currentRepo)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LastModified.Equal(items[j].LastModified) {
			return items[i].DisplayName < items[j].DisplayName
		}
		return items[i].LastModified.After(items[j].LastModified)
	})
	return items, nil
}

func resolveManagedRepo(root, encodedRepo string) (*managedRepo, error) {
	repos, err := discoverManagedRepos(root)
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		if repo.EncodedPath == encodedRepo {
			copy := repo
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("repo not found")
}

func validateLocalGitRepo(repoPath string) error {
	if strings.TrimSpace(repoPath) == "" {
		return fmt.Errorf("repository path is empty")
	}
	if isLikelyGitURL(repoPath) {
		return nil
	}
	runner := gitcmd.New(2)
	out, err := runner.Output(context.Background(), repoPath, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("repository path is not a valid git repository: %s", repoPath)
	}
	return nil
}

func handleRepos(w http.ResponseWriter, r *http.Request, root string) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	repos, err := DiscoverRepos(root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "repos.html", map[string]any{
		"Repos": repos,
	})
}

func handleReposAPI(w http.ResponseWriter, _ *http.Request, root string) {
	reg, err := readViewerRepoRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	tokenReg, _ := readRepoTokenRegistry()
	items := make([]repoListItem, 0, len(reg.Repos))
	for _, entry := range reg.Repos {
		encoded := encodeRepoPath(entry.LocalPath)
		summaries, _ := ListSessions(root, encoded)
		hasToken := entry.Token != "" || tokenReg.Tokens[encoded] != ""
		items = append(items, repoListItem{
			EncodedPath:  encoded,
			DisplayName:  entry.Name,
			RepoPath:     entry.LocalPath,
			RemoteURL:    entry.RemoteURL,
			SessionCount: len(summaries),
			FileCount:    entry.FileCount,
			LastModified: "",
			HasToken:     hasToken,
		})
	}

	writeJSON(w, http.StatusOK, repoListResponse{Repos: items})
}

func handleSyncRepoAPI(w http.ResponseWriter, r *http.Request, root string, repoName string) {
	reg, err := readViewerRepoRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var targetEntry *viewerRepoEntry
	var targetIndex int
	for i, entry := range reg.Repos {
		if repoEntryMatches(entry, repoName) {
			targetEntry = &entry
			targetIndex = i
			break
		}
	}

	if targetEntry == nil || targetEntry.RemoteURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "没有远程地址"})
		return
	}

	cloneDir := targetEntry.LocalPath

	tokenReg, _ := readRepoTokenRegistry()
	cloneURL := targetEntry.RemoteURL
	encoded := repoName

	encURL := urlRepoPrefix + targetEntry.RemoteURL
	if _, hasToken := tokenReg.Tokens[encoded]; hasToken {
		u, err := url.Parse(targetEntry.RemoteURL)
		if err == nil {
			u.User = url.User(tokenReg.Tokens[encoded])
			cloneURL = u.String()
		}
	} else if _, hasToken := tokenReg.Tokens[encURL]; hasToken {
		u, err := url.Parse(targetEntry.RemoteURL)
		if err == nil {
			u.User = url.User(tokenReg.Tokens[encURL])
			cloneURL = u.String()
		}
	}

	if err := os.MkdirAll(filepath.Dir(cloneDir), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create clone dir"})
		return
	}

	runner := gitcmd.New(2)
	if _, statErr := os.Stat(filepath.Join(cloneDir, ".git")); statErr == nil {
		_, err = runner.Run(context.Background(), cloneDir, "pull", "--ff-only")
	} else {
		_, err = runner.Run(context.Background(), filepath.Dir(cloneDir), "clone", "--depth", "1", cloneURL, cloneDir)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("同步失败：%v", err)})
		return
	}

	fileCount := countRepoFiles(cloneDir)

	targetEntry.LocalPath = cloneDir
	targetEntry.FileCount = fileCount
	reg.Repos[targetIndex] = *targetEntry

	if err := writeViewerRepoRegistry(reg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存失败"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"localPath": cloneDir,
		"fileCount": fileCount,
	})
}

func handleDeleteRepoAPI(w http.ResponseWriter, r *http.Request, root string, repoName string) {
	reg, err := readViewerRepoRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	newRepos := make([]viewerRepoEntry, 0)
	var removedEntry *viewerRepoEntry
	for _, entry := range reg.Repos {
		if !repoEntryMatches(entry, repoName) {
			newRepos = append(newRepos, entry)
		} else {
			copy := entry
			removedEntry = &copy
		}
	}

	if len(newRepos) == len(reg.Repos) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "仓库不存在"})
		return
	}

	reg.Repos = newRepos
	if err := writeViewerRepoRegistry(reg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	sessionPath := filepath.Join(root, repoName)
	os.RemoveAll(sessionPath)

	if removedEntry != nil && strings.Contains(removedEntry.LocalPath, filepath.Join(".opencodereview", "viewer_sync")) {
		os.RemoveAll(removedEntry.LocalPath)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func repoEntryMatches(entry viewerRepoEntry, encodedRepo string) bool {
	return encodeRepoPath(entry.LocalPath) == encodedRepo || encodeViewerRepoName(entry.Name) == encodedRepo
}

const clonedReposDir = "cloned_repos"
const urlRepoPrefix = "__url__"

func urlToDirName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		sanitized := strings.NewReplacer(":", "-", "/", "-", "@", "-").Replace(rawURL)
		return sanitized
	}

	host := u.Host
	if host == "" {
		host = "local"
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	if path == "" {
		path = "repo"
	}

	return host + "-" + strings.ReplaceAll(path, "/", "-")
}

func readDirCount(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			count++
		}
	}
	return count
}

func isLikelyGitURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "git@")
}

type repoTokenRegistry struct {
	Tokens map[string]string `json:"tokens"`
}

func readRepoTokenRegistry() (*repoTokenRegistry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".opencodereview", "repo_tokens.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &repoTokenRegistry{Tokens: make(map[string]string)}, nil
		}
		return nil, err
	}
	var reg repoTokenRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	if reg.Tokens == nil {
		reg.Tokens = make(map[string]string)
	}
	return &reg, nil
}

func writeRepoTokenRegistry(reg *repoTokenRegistry) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".opencodereview")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "repo_tokens.json")
	data, err := json.MarshalIndent(reg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func cloneRepoToManagedDir(repoURL string) (string, error) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repo URL: %w", err)
	}
	name := strings.TrimSuffix(path.Base(parsed.Path), ".git")
	if name == "" || name == "." || name == "/" {
		name = strings.TrimSuffix(filepath.Base(repoURL), ".git")
	}
	if strings.Contains(name, "/") {
		name = strings.ReplaceAll(name, "/", "-")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	managedDir := filepath.Join(home, ".opencodereview", clonedReposDir)
	targetPath := filepath.Join(managedDir, name)

	_, err = os.Stat(targetPath)
	if err == nil {
		runner := gitcmd.New(30)
		_, err := runner.Run(context.Background(), targetPath, "pull", "--ff-only")
		if err == nil {
			return targetPath, nil
		}
		os.RemoveAll(targetPath)
	}

	runner := gitcmd.New(300)
	_, err = runner.Run(context.Background(), managedDir, "clone", "--depth", "1", repoURL, name)
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w", err)
	}
	return targetPath, nil
}

func handleAddRepoAPI(w http.ResponseWriter, r *http.Request, root string) {
	var req addRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	repoName := strings.TrimSpace(req.RepoName)
	if repoName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "仓库名称是必填的"})
		return
	}
	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "本地地址是必填的"})
		return
	}

	viewerRepoStore.Lock()
	defer viewerRepoStore.Unlock()
	reg, err := readViewerRepoRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	for _, existing := range reg.Repos {
		if existing.Name == repoName {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "仓库名称已存在"})
			return
		}
	}

	reg.Repos = append(reg.Repos, viewerRepoEntry{
		Name:      repoName,
		LocalPath: localPath,
		RemoteURL: req.RemoteURL,
		Token:     req.Token,
	})
	if err := writeViewerRepoRegistry(reg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, repoListItem{
		EncodedPath:  pathToEncoded(repoName),
		DisplayName:  repoName,
		RepoPath:     localPath,
		RemoteURL:    req.RemoteURL,
		SessionCount: 0,
	})
}

func handleUpdateRepoTokenAPI(w http.ResponseWriter, r *http.Request, encodedRepo string) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	tokenReg, err := readRepoTokenRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if req.Token == "" {
		delete(tokenReg.Tokens, encodedRepo)
	} else {
		tokenReg.Tokens[encodedRepo] = req.Token
	}
	if err := writeRepoTokenRegistry(tokenReg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func urlDisplayName(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	name := path.Base(parsed.Path)
	name = strings.TrimSuffix(name, ".git")
	if name == "" || name == "." {
		name = rawURL
	}
	return name
}

func pathToEncoded(p string) string {
	return strings.ReplaceAll(p, "/", "_")
}

type sessionsData struct {
	EncodedRepo string
	RepoName    string
	Sessions    []SessionSummary
}

func handleSessions(w http.ResponseWriter, r *http.Request, root, repo string) {
	summaries, err := ListSessions(root, repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Derive a display name from the first session's CWD
	name := deriveRepoName(repo, summaries)

	renderTemplate(w, "sessions.html", sessionsData{
		EncodedRepo: repo,
		RepoName:    name,
		Sessions:    summaries,
	})
}

func handleSessionsAPI(w http.ResponseWriter, _ *http.Request, root, repo string) {
	summaries, err := ListSessions(root, repo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, sessionListResponse{
		EncodedRepo: repo,
		RepoName:    deriveRepoName(repo, summaries),
		Sessions:    summaries,
	})
}

func handleAllSessionsAPI(w http.ResponseWriter, _ *http.Request, root string) {
	repos, err := discoverManagedRepos(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var allSessions []SessionSummary
	for _, repo := range repos {
		summaries, err := ListSessions(root, repo.EncodedPath)
		if err == nil {
			for _, s := range summaries {
				s.RepoName = repo.DisplayName
				s.RepoPath = repo.RepoPath
				allSessions = append(allSessions, s)
			}
		}
	}

	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].Timestamp.After(allSessions[j].Timestamp)
	})

	writeJSON(w, http.StatusOK, map[string][]SessionSummary{"sessions": allSessions})
}

func handleCloneRepoAPI(w http.ResponseWriter, r *http.Request, root string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		EncodedRepo string `json:"encodedRepo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	repos, err := discoverManagedRepos(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var repoPath string
	for _, repo := range repos {
		if repo.EncodedPath == req.EncodedRepo {
			repoPath = repo.RepoPath
			break
		}
	}

	if repoPath == "" || !isLikelyGitURL(repoPath) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a remote repository"})
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get home dir"})
		return
	}

	cloneDir := filepath.Join(home, ".opencodereview", clonedReposDir, "remote", urlToDirName(repoPath))

	// 删除旧目录，确保每次都是全新克隆
	if _, err := os.Stat(cloneDir); err == nil {
		if err := os.RemoveAll(cloneDir); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to remove old clone: %v", err)})
			return
		}
	}

	// 创建父目录
	if err := os.MkdirAll(filepath.Dir(cloneDir), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to create clone dir: %v", err)})
		return
	}

	tokenReg, _ := readRepoTokenRegistry()
	_, hasToken := tokenReg.Tokens[req.EncodedRepo]

	cloneURL := repoPath
	if hasToken {
		u, err := url.Parse(repoPath)
		if err == nil {
			u.User = url.User(tokenReg.Tokens[req.EncodedRepo])
			cloneURL = u.String()
		}
	}

	runner := gitcmd.New(2)
	_, err = runner.Run(context.Background(), filepath.Dir(cloneDir), "clone", "--bare", cloneURL, cloneDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to clone: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cloned", "cloneDir": cloneDir})
}

func handleCheckCloneStatusAPI(w http.ResponseWriter, r *http.Request, root string) {
	var req struct {
		EncodedRepo string `json:"encodedRepo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	repos, err := discoverManagedRepos(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var repoPath string
	for _, repo := range repos {
		if repo.EncodedPath == req.EncodedRepo {
			repoPath = repo.RepoPath
			break
		}
	}

	if repoPath == "" || !isLikelyGitURL(repoPath) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"cloned": false, "isRemote": true})
		return
	}

	home, _ := os.UserHomeDir()
	cloneDir := filepath.Join(home, ".opencodereview", clonedReposDir, "remote", urlToDirName(repoPath))

	_, err = os.Stat(cloneDir)
	cloned := err == nil

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"cloned":   cloned,
		"isRemote": true,
		"cloneDir": cloneDir,
	})
}

type sessionPageData struct {
	EncodedRepo string
	RepoName    string
	Session     *ViewSession
}

func handleSession(w http.ResponseWriter, r *http.Request, root, repo, sessionID string) {
	vs, err := LoadSession(root, repo, sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load session: %v", err), http.StatusNotFound)
		return
	}

	// Derive a display name
	name := filepath.Base(vs.Summary.CWD)
	if name == "." || name == "" {
		name = repo
	}

	renderTemplate(w, "session.html", sessionPageData{
		EncodedRepo: repo,
		RepoName:    name,
		Session:     vs,
	})
}

func handleSessionAPI(w http.ResponseWriter, _ *http.Request, root, repo, sessionID string) {
	vs, err := LoadSession(root, repo, sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("failed to load session: %v", err)})
		return
	}

	name := filepath.Base(vs.Summary.CWD)
	if name == "." || name == "" {
		name = repo
	}

	writeJSON(w, http.StatusOK, sessionDetailResponse{
		EncodedRepo: repo,
		RepoName:    name,
		Session:     vs,
	})
}

func handleDeleteSessionAPI(w http.ResponseWriter, _ *http.Request, root, repo, sessionID string) {
	vs, err := LoadSession(root, repo, sessionID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("failed to load session: %v", err)})
		return
	}
	if vs.Summary.Status == "running" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "running sessions cannot be deleted"})
		return
	}

	if err := os.Remove(filepath.Join(root, repo, sessionID+".jsonl")); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleRulesAPI(w http.ResponseWriter, _ *http.Request, root string) {
	systemRule, err := ruleconfig.LoadDefault()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	home := ""
	if userHome, err := os.UserHomeDir(); err == nil {
		home = userHome
	}
	globalPath := "~/.opencodereview/rule.json"
	if home != "" {
		globalPath = filepath.Join(home, ".opencodereview", "rule.json")
	}

	layers := []ruleLayerSummary{
		{
			Priority:    1,
			Source:      "custom",
			Title:       "--rule flag",
			Path:        "Specified when creating a review task",
			Description: "Highest priority. Applies only to review tasks that pass a custom rule file path.",
			Available:   false,
		},
		{
			Priority:    2,
			Source:      "project",
			Title:       "Project rule files",
			Path:        "<repoDir>/.opencodereview/rule.json",
			Description: "Per-repository rules discovered from registered local repositories.",
			Available:   false,
		},
		{
			Priority:    3,
			Source:      "global",
			Title:       "Global rule file",
			Path:        globalPath,
			Description: "User-wide rule file loaded from the current account.",
			Available:   false,
		},
		{
			Priority:    4,
			Source:      "system",
			Title:       "System defaults",
			Path:        "embedded:system_rules.json",
			Description: "Built-in fallback rules used when higher-priority layers do not match.",
			Available:   true,
			DefaultRule: systemRule.DefaultRule,
			PathRules:   make([]systemRuleEntry, 0, len(systemRule.PathRules)),
		},
	}

	for _, entry := range systemRule.PathRules {
		layers[3].PathRules = append(layers[3].PathRules, systemRuleEntry{
			Pattern: entry.Pattern,
			Rule:    entry.Rule,
		})
	}

	if home != "" {
		if data, err := os.ReadFile(globalPath); err == nil {
			var globalRule ruleconfig.ProjectRule
			if json.Unmarshal(data, &globalRule) == nil {
				layers[2].Available = true
				layers[2].Rules = globalRule.Rules
				layers[2].Include = globalRule.Include
				layers[2].Exclude = globalRule.Exclude
			}
		}
	}

	projectRules := make([]ruleLayerSummary, 0)
	repos, err := discoverManagedRepos(root)
	if err == nil {
		for _, repo := range repos {
			if repo.RepoPath == "" {
				continue
			}
			projectPath := filepath.Join(repo.RepoPath, ".opencodereview", "rule.json")
			data, err := os.ReadFile(projectPath)
			if err != nil {
				continue
			}
			var projectRule ruleconfig.ProjectRule
			if json.Unmarshal(data, &projectRule) != nil {
				continue
			}
			projectRules = append(projectRules, ruleLayerSummary{
				Priority:    2,
				Source:      fmt.Sprintf("project:%s", repo.DisplayName),
				Title:       repo.DisplayName,
				Path:        projectPath,
				Description: "Project-local rule file for this repository.",
				Available:   true,
				Rules:       projectRule.Rules,
				Include:     projectRule.Include,
				Exclude:     projectRule.Exclude,
			})
		}
	}
	if len(projectRules) > 0 {
		layers[1].Available = true
		layers[1].Path = fmt.Sprintf("%d project rule file(s)", len(projectRules))
		layers = append(layers[:2], append(projectRules, layers[2:]...)...)
	}

	writeJSON(w, http.StatusOK, rulesOverviewResponse{Layers: layers})
}

func handleRepoStatusAPI(w http.ResponseWriter, _ *http.Request, root, repo string) {
	repoInfo, err := resolveManagedRepo(root, repo)
	if err != nil || repoInfo.RepoPath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository path not found"})
		return
	}

	repoPath := repoInfo.RepoPath
	runner := gitcmd.New(4)
	branchOut, err := runner.Run(context.Background(), repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	headOut, err := runner.Run(context.Background(), repoPath, "rev-parse", "--short", "HEAD")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	statusOut, err := runner.Run(context.Background(), repoPath, "status", "--short")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	stats := repoStatusStats{}
	files := make([]repoFileStatus, 0)
	for _, line := range strings.Split(strings.TrimSpace(statusOut), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || len(line) < 3 {
			continue
		}
		indexStatus := string(line[0])
		worktreeStatus := string(line[1])
		path := strings.TrimSpace(line[3:])
		category := "modified"

		if indexStatus == "?" && worktreeStatus == "?" {
			stats.Untracked++
			category = "untracked"
		} else {
			if indexStatus != " " {
				stats.Staged++
			}
			if worktreeStatus != " " {
				stats.Unstaged++
			}
			if indexStatus == "U" || worktreeStatus == "U" || (indexStatus == "A" && worktreeStatus == "A") {
				stats.Conflicts++
				category = "conflict"
			} else if indexStatus == "A" || worktreeStatus == "A" {
				category = "added"
			} else if indexStatus == "D" || worktreeStatus == "D" {
				category = "deleted"
			}
		}

		files = append(files, repoFileStatus{
			Path:           path,
			IndexStatus:    indexStatus,
			WorktreeStatus: worktreeStatus,
			Category:       category,
		})
	}
	stats.Total = len(files)

	writeJSON(w, http.StatusOK, repoStatusResponse{
		EncodedRepo: repo,
		RepoName:    repoInfo.DisplayName,
		RepoPath:    repoPath,
		Branch:      strings.TrimSpace(branchOut),
		Head:        strings.TrimSpace(headOut),
		Stats:       stats,
		Files:       files,
	})
}

func launchReviewTask(task *reviewTaskResponse) {
	exe, err := os.Executable()
	if err != nil {
		reviewTaskStore.Lock()
		task.Status = "failed"
		task.ErrorMessage = err.Error()
		task.FinishedAt = time.Now()
		reviewTaskStore.Unlock()
		return
	}

	runner := gitcmd.New(2)
	workDir := task.RepoPath
	cleanup := false

	fmt.Printf("[launchReviewTask] Start: encodedRepo=%s, repoPath=%s, branch=%s\n",
		task.EncodedRepo, task.RepoPath, task.Branch)

	if isLikelyGitURL(task.RepoPath) {
		home, err := os.UserHomeDir()
		if err != nil {
			reviewTaskStore.Lock()
			task.Status = "failed"
			task.ErrorMessage = fmt.Sprintf("failed to get home dir: %v", err)
			task.FinishedAt = time.Now()
			reviewTaskStore.Unlock()
			return
		}

		cloneDir := filepath.Join(home, ".opencodereview", clonedReposDir, "remote", urlToDirName(task.RepoPath))
		cloneDirExists := false
		if _, statErr := os.Stat(cloneDir); statErr == nil {
			cloneDirExists = true
		}
		fmt.Printf("[launchReviewTask] cloneDir=%s, exists=%v\n", cloneDir, cloneDirExists)

		tokenReg, _ := readRepoTokenRegistry()
		_, hasToken := tokenReg.Tokens[task.EncodedRepo]

		cloneURL := task.RepoPath
		if hasToken {
			u, err := url.Parse(task.RepoPath)
			if err == nil {
				u.User = url.User(tokenReg.Tokens[task.EncodedRepo])
				cloneURL = u.String()
			}
		}

		if _, err := os.Stat(cloneDir); os.IsNotExist(err) {
			_, err := runner.Run(context.Background(), filepath.Dir(cloneDir), "clone", "--bare", cloneURL, cloneDir)
			if err != nil {
				reviewTaskStore.Lock()
				task.Status = "failed"
				task.ErrorMessage = fmt.Sprintf("failed to clone repository: %v", err)
				task.FinishedAt = time.Now()
				reviewTaskStore.Unlock()
				return
			}
		} else {
			_, err := runner.Run(context.Background(), cloneDir, "fetch", "--all", "--prune")
			if err != nil {
				reviewTaskStore.Lock()
				task.Status = "failed"
				task.ErrorMessage = fmt.Sprintf("failed to fetch updates: %v", err)
				task.FinishedAt = time.Now()
				reviewTaskStore.Unlock()
				return
			}
		}

		workDir, err = os.MkdirTemp("", "git-review-*")
		if err != nil {
			reviewTaskStore.Lock()
			task.Status = "failed"
			task.ErrorMessage = fmt.Sprintf("failed to create temp dir: %v", err)
			task.FinishedAt = time.Now()
			reviewTaskStore.Unlock()
			return
		}
		cleanup = true

		// 删除临时目录，直接从 bare repo clone 到这个路径
		os.RemoveAll(workDir)
		_, err = runner.Run(context.Background(), filepath.Dir(workDir), "clone", cloneDir, workDir)
		if err != nil {
			os.RemoveAll(workDir)
			reviewTaskStore.Lock()
			task.Status = "failed"
			task.ErrorMessage = fmt.Sprintf("failed to clone from bare repo: %v", err)
			task.FinishedAt = time.Now()
			reviewTaskStore.Unlock()
			return
		}
		fmt.Printf("[launchReviewTask] cloned to workDir=%s, files=%v\n", workDir, readDirCount(workDir))

		if task.Branch != "" {
			_, err := runner.Run(context.Background(), workDir, "checkout", task.Branch)
			if err != nil {
				_, fetchErr := runner.Run(context.Background(), workDir, "checkout", "-b", task.Branch, "origin/"+task.Branch)
				if fetchErr != nil {
					os.RemoveAll(workDir)
					reviewTaskStore.Lock()
					task.Status = "failed"
					task.ErrorMessage = fmt.Sprintf("failed to checkout branch %s: %v", task.Branch, err)
					task.FinishedAt = time.Now()
					reviewTaskStore.Unlock()
					return
				}
			}
		}
	} else {
		if task.Branch != "" {
			_, err := runner.Run(context.Background(), task.RepoPath, "checkout", task.Branch)
			if err != nil {
				reviewTaskStore.Lock()
				task.Status = "failed"
				task.ErrorMessage = fmt.Sprintf("failed to checkout branch %s: %v", task.Branch, err)
				task.FinishedAt = time.Now()
				reviewTaskStore.Unlock()
				return
			}
		}
	}

	if cleanup {
		defer os.RemoveAll(workDir)
	}

	fmt.Printf("[launchReviewTask] workDir=%s, branch=%s, mode=%s, from=%s, to=%s, commit=%s\n",
		workDir, task.Branch, task.ReviewMode, task.BaseRef, task.TargetRef, task.CommitRef)

	args := []string{"review", "--repo", workDir, "--audience", "agent"}

	switch task.ReviewMode {
	case "full":
		args = append(args, "--full")
	case "branch-diff":
		if task.BaseRef != "" && task.TargetRef != "" {
			args = append(args, "--from", task.BaseRef, "--to", task.TargetRef)
		}
	case "commit":
		if task.CommitRef != "" {
			args = append(args, "--commit", task.CommitRef)
		}
	}

	if task.Background != "" {
		args = append(args, "--background", task.Background)
	}
	if task.Format != "" && task.Format != "text" {
		args = append(args, "--format", task.Format)
	}
	if task.Timeout != "" {
		args = append(args, "--timeout", task.Timeout)
	}
	if task.Concurrency != "" {
		args = append(args, "--concurrency", task.Concurrency)
	}
	if task.RulePath != "" {
		args = append(args, "--rule", task.RulePath)
	}

	cmd := exec.Command(exe, args...)
	cmd.Dir = workDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	reviewTaskStore.Lock()
	defer reviewTaskStore.Unlock()
	task.FinishedAt = time.Now()
	task.Output = strings.TrimSpace(stdout.String())
	if err != nil {
		task.Status = "failed"
		task.ErrorMessage = strings.TrimSpace(stderr.String())
		if task.ErrorMessage == "" {
			task.ErrorMessage = err.Error()
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			task.ExitCode = exitErr.ExitCode()
		}
		return
	}
	task.Status = "completed"
	if stderr.Len() > 0 {
		task.ErrorMessage = strings.TrimSpace(stderr.String())
	}
}

func handleCreateReviewTaskAPI(w http.ResponseWriter, r *http.Request, root string) {
	var req createReviewTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	repoInfo, err := resolveManagedRepo(root, req.EncodedRepo)
	if err != nil || repoInfo.RepoPath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository path not found"})
		return
	}
	if err := validateLocalGitRepo(repoInfo.RepoPath); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	taskID := fmt.Sprintf("review-%d", time.Now().UnixNano())
	task := &reviewTaskResponse{
		TaskID:      taskID,
		EncodedRepo: repoInfo.EncodedPath,
		RepoName:    repoInfo.DisplayName,
		RepoPath:    repoInfo.RepoPath,
		Branch:      req.EncodedBranch,
		ReviewMode:  req.ReviewMode,
		BaseRef:     req.BaseRef,
		TargetRef:   req.TargetRef,
		CommitRef:   req.CommitRef,
		Background:  req.Background,
		Format:      req.Format,
		Timeout:     req.Timeout,
		Concurrency: req.Concurrency,
		RulePath:    req.RulePath,
		Status:      "running",
		StartedAt:   time.Now(),
	}
	reviewTaskStore.Lock()
	reviewTaskStore.tasks[taskID] = task
	reviewTaskStore.Unlock()
	go launchReviewTask(task)
	writeJSON(w, http.StatusAccepted, task)
}

func handleReviewTaskAPI(w http.ResponseWriter, _ *http.Request, taskID string) {
	reviewTaskStore.Lock()
	defer reviewTaskStore.Unlock()
	task, ok := reviewTaskStore.tasks[taskID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func handleListBranchesAPI(w http.ResponseWriter, r *http.Request, root string) {
	encodedRepo := r.URL.Query().Get("repo")
	if encodedRepo == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo parameter required"})
		return
	}

	repos, err := discoverManagedRepos(root)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var repoPath string
	for _, repo := range repos {
		if repo.EncodedPath == encodedRepo {
			repoPath = repo.RepoPath
			break
		}
	}

	if repoPath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "repository not found"})
		return
	}

	runner := gitcmd.New(2)

	if isLikelyGitURL(repoPath) {
		tmpDir, err := os.MkdirTemp("", "git-clone-*")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create temp dir"})
			return
		}
		defer os.RemoveAll(tmpDir)

		tokenReg, _ := readRepoTokenRegistry()
		_, hasToken := tokenReg.Tokens[encodedRepo]

		cloneURL := repoPath
		if hasToken {
			u, err := url.Parse(repoPath)
			if err == nil {
				u.User = url.User(tokenReg.Tokens[encodedRepo])
				cloneURL = u.String()
			}
		}

		_, err = runner.Run(context.Background(), tmpDir, "clone", "--no-checkout", "--depth", "1", cloneURL, tmpDir)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to clone repository"})
			return
		}
		repoPath = tmpDir
	}

	out, err := runner.Output(context.Background(), repoPath, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list branches"})
		return
	}

	branches := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(branches) == 1 && branches[0] == "" {
		branches = []string{}
	}

	writeJSON(w, http.StatusOK, map[string][]string{"branches": branches})
}
