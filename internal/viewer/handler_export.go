package viewer

import "net/http"

// Exported handler functions for use in serve command

// HandleReposAPI handles GET /api/repos
func HandleReposAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleReposAPI(w, r, root)
}

// HandleAddRepoAPI handles POST /api/repos
func HandleAddRepoAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleAddRepoAPI(w, r, root)
}

// HandleUpdateRepoTokenAPI handles PUT /api/repos/{repo}/token
func HandleUpdateRepoTokenAPI(w http.ResponseWriter, r *http.Request, repo string) {
	handleUpdateRepoTokenAPI(w, r, repo)
}

// HandleSyncRepoAPI handles POST /api/repos/{repo}/sync
func HandleSyncRepoAPI(w http.ResponseWriter, r *http.Request, root, repoName string) {
	handleSyncRepoAPI(w, r, root, repoName)
}

// HandleDeleteRepoAPI handles DELETE /api/repos/{repo}
func HandleDeleteRepoAPI(w http.ResponseWriter, r *http.Request, root, repoName string) {
	handleDeleteRepoAPI(w, r, root, repoName)
}

// HandleSessionsAPI handles GET /api/repos/{repo}/sessions
func HandleSessionsAPI(w http.ResponseWriter, r *http.Request, root, repo string) {
	handleSessionsAPI(w, r, root, repo)
}

// HandleSessionAPI handles GET /api/repos/{repo}/sessions/{sessionID}
func HandleSessionAPI(w http.ResponseWriter, r *http.Request, root, repo, sessionID string) {
	handleSessionAPI(w, r, root, repo, sessionID)
}

// HandleDeleteSessionAPI handles DELETE /api/repos/{repo}/sessions/{sessionID}
func HandleDeleteSessionAPI(w http.ResponseWriter, r *http.Request, root, repo, sessionID string) {
	handleDeleteSessionAPI(w, r, root, repo, sessionID)
}

// HandleGenerateAuditReportAPI handles POST /api/repos/{repo}/sessions/{sessionID}/report
func HandleGenerateAuditReportAPI(w http.ResponseWriter, r *http.Request, root, repo, sessionID string) {
	handleGenerateAuditReportAPI(w, r, root, repo, sessionID)
}

// HandleRepoStatusAPI handles GET /api/repos/{repo}/status
func HandleRepoStatusAPI(w http.ResponseWriter, r *http.Request, root, repo string) {
	handleRepoStatusAPI(w, r, root, repo)
}

// HandleAllSessionsAPI handles GET /api/sessions
func HandleAllSessionsAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleAllSessionsAPI(w, r, root)
}

// HandleCreateReviewTaskAPI handles POST /api/reviews
func HandleCreateReviewTaskAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleCreateReviewTaskAPI(w, r, root)
}

// HandleReviewTaskAPI handles GET /api/reviews/{taskID}
func HandleReviewTaskAPI(w http.ResponseWriter, r *http.Request, taskID string) {
	handleReviewTaskAPI(w, r, taskID)
}

// HandleLLMConfigAPI handles GET /api/config/llm
func HandleLLMConfigAPI(w http.ResponseWriter, r *http.Request) {
	handleLLMConfigAPI(w, r)
}

// HandleSaveLLMConfigAPI handles POST /api/config/llm
func HandleSaveLLMConfigAPI(w http.ResponseWriter, r *http.Request) {
	handleSaveLLMConfigAPI(w, r)
}

// HandleRulesAPI handles GET /api/rules
func HandleRulesAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleRulesAPI(w, r, root)
}

// HandleListAuditReportsAPI handles GET /api/reports
func HandleListAuditReportsAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleListAuditReportsAPI(w, r, root)
}

// HandleGetAuditReportAPI handles GET /api/reports/{reportID}
func HandleGetAuditReportAPI(w http.ResponseWriter, r *http.Request, root, reportID string) {
	handleGetAuditReportAPI(w, r, root, reportID)
}

// HandleListBranchesAPI handles GET /api/branches
func HandleListBranchesAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleListBranchesAPI(w, r, root)
}

// HandleGenerateIssueListAPI handles POST /api/repos/{repo}/sessions/{sessionID}/issues
func HandleGenerateIssueListAPI(w http.ResponseWriter, r *http.Request, root, repo, sessionID string) {
	handleGenerateIssueListAPI(w, r, root, repo, sessionID)
}

// HandleGetIssueListAPI handles GET /api/repos/{repo}/sessions/{sessionID}/issues or GET /api/issues/{issueListID}
func HandleGetIssueListAPI(w http.ResponseWriter, r *http.Request, root, repo, sessionID string) {
	handleGetIssueListAPI(w, r, root, repo, sessionID)
}

// HandleListIssueListsAPI handles GET /api/issues
func HandleListIssueListsAPI(w http.ResponseWriter, r *http.Request, root string) {
	handleListIssueListsAPI(w, r, root)
}

// ValidatePathValue validates a path value
func ValidatePathValue(p string) bool {
	return validatePathValue(p)
}

// ResolveAllowedHostsFromEnv resolves allowed hosts from environment
func ResolveAllowedHostsFromEnv(addr string) map[string]struct{} {
	return resolveAllowedHostsFromEnv(addr)
}

// HostGuard creates a host guard wrapper
func HostGuard(allowed map[string]struct{}, next http.Handler) http.Handler {
	return hostGuard(allowed, next)
}
