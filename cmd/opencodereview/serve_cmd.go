package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/open-code-review/open-code-review/internal/static"
	"github.com/open-code-review/open-code-review/internal/viewer"
)

type serveOptions struct {
	addr     string
	showHelp bool
}

func parseServeFlags(args []string) (serveOptions, error) {
	a := newOcrFlagSet("ocr serve")

	opts := serveOptions{}
	a.StringVar(&opts.addr, "addr", "localhost:3030", "listen address")

	if err := a.Parse(args); err != nil {
		return opts, fmt.Errorf("parse flags: %w", err)
	}

	opts.showHelp = a.showHelp
	return opts, nil
}

func runServe(args []string) error {
	opts, err := parseServeFlags(args)
	if err != nil {
		return err
	}
	if opts.showHelp {
		printServeUsage()
		return nil
	}

	root, err := viewer.SessionsRoot()
	if err != nil {
		return fmt.Errorf("resolve sessions root: %w", err)
	}

	// Create API mux
	apiMux := createAPIMux(root)

	// Create static file server for frontend
	staticFS, err := fs.Sub(static.WebFS, "dist")
	if err != nil {
		return fmt.Errorf("create static FS: %w", err)
	}
	staticServer := http.FileServer(http.FS(staticFS))

	// Combined mux: /api/* → API, /* → static files (SPA fallback)
	mux := http.NewServeMux()
	mux.Handle("/api/", apiMux)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA: all non-API routes serve index.html
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			r.URL.Path = "/index.html"
		}
		staticServer.ServeHTTP(w, r)
	})

	// Host guard for security
	allowed := viewer.ResolveAllowedHostsFromEnv(opts.addr)
	guarded := viewer.HostGuard(allowed, mux)

	srv := &http.Server{
		Addr:    opts.addr,
		Handler: guarded,
	}

	fmt.Printf("OpenCodeReview Web Console starting on http://%s\n", opts.addr)
	return srv.ListenAndServe()
}

func createAPIMux(root string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/repos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			viewer.HandleReposAPI(w, r, root)
		case http.MethodPost:
			viewer.HandleAddRepoAPI(w, r, root)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/repos/{repo}/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !viewer.ValidatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		viewer.HandleUpdateRepoTokenAPI(w, r, repo)
	})
	mux.HandleFunc("/api/repos/{repo}/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !viewer.ValidatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		viewer.HandleSyncRepoAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/repos/{repo}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !viewer.ValidatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		viewer.HandleDeleteRepoAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !viewer.ValidatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		viewer.HandleSessionsAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions/{sessionID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		sid := r.PathValue("sessionID")
		if !viewer.ValidatePathValue(repo) || !viewer.ValidatePathValue(sid) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			viewer.HandleDeleteSessionAPI(w, r, root, repo, sid)
			return
		}
		viewer.HandleSessionAPI(w, r, root, repo, sid)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions/{sessionID}/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		sid := r.PathValue("sessionID")
		if !viewer.ValidatePathValue(repo) || !viewer.ValidatePathValue(sid) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		viewer.HandleGenerateAuditReportAPI(w, r, root, repo, sid)
	})
	mux.HandleFunc("/api/repos/{repo}/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !viewer.ValidatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		viewer.HandleRepoStatusAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		viewer.HandleAllSessionsAPI(w, r, root)
	})
	mux.HandleFunc("/api/reviews", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		viewer.HandleCreateReviewTaskAPI(w, r, root)
	})
	mux.HandleFunc("/api/reviews/{taskID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		taskID := r.PathValue("taskID")
		if !viewer.ValidatePathValue(taskID) && !strings.HasPrefix(taskID, "review-") {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}
		viewer.HandleReviewTaskAPI(w, r, taskID)
	})
	mux.HandleFunc("/api/config/llm", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			viewer.HandleLLMConfigAPI(w, r)
		case http.MethodPost:
			viewer.HandleSaveLLMConfigAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		viewer.HandleRulesAPI(w, r, root)
	})
	mux.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		viewer.HandleListAuditReportsAPI(w, r, root)
	})
	mux.HandleFunc("/api/reports/{reportID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reportID := r.PathValue("reportID")
		if !viewer.ValidatePathValue(reportID) {
			http.Error(w, "invalid report id", http.StatusBadRequest)
			return
		}
		viewer.HandleGetAuditReportAPI(w, r, root, reportID)
	})
	mux.HandleFunc("/api/branches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		viewer.HandleListBranchesAPI(w, r, root)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions/{sessionID}/issues", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			repo := r.PathValue("repo")
			sid := r.PathValue("sessionID")
			if !viewer.ValidatePathValue(repo) || !viewer.ValidatePathValue(sid) {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			viewer.HandleGenerateIssueListAPI(w, r, root, repo, sid)
		case http.MethodGet:
			repo := r.PathValue("repo")
			sid := r.PathValue("sessionID")
			if !viewer.ValidatePathValue(repo) || !viewer.ValidatePathValue(sid) {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}
			viewer.HandleGetIssueListAPI(w, r, root, repo, sid)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/issues", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		viewer.HandleListIssueListsAPI(w, r, root)
	})
	mux.HandleFunc("/api/issues/{issueListID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		issueListID := r.PathValue("issueListID")
		if !viewer.ValidatePathValue(issueListID) {
			http.Error(w, "invalid issue list id", http.StatusBadRequest)
			return
		}
		viewer.HandleGetIssueListAPI(w, r, root, "", issueListID)
	})

	return mux
}

func printServeUsage() {
	fmt.Println(`OpenCodeReview Web Console (Production Mode).

Usage:
  ocr serve [flags]

Flags:
  --addr <address>           listen address (default: localhost:3030)

Examples:
  ocr serve                      # start on localhost:3030
  ocr serve --addr :3030         # bind to all interfaces
  ocr serve --addr 0.0.0.0:3030  # same as above

The web console provides:
  - Repository management
  - Full repository audit mode
  - Task list and session history
  - Issue tracking and audit reports
  - LLM configuration
  - Rule management`)
}
