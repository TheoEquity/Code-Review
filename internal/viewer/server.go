package viewer

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed templates/*.html static/style.css
var assets embed.FS

func StartServer(addr string) error {
	root, err := SessionsRoot()
	if err != nil {
		return fmt.Errorf("resolve sessions root: %w", err)
	}

	mux := http.NewServeMux()

	// Static assets (must be registered before "/" catch-all)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS()))))
	mux.HandleFunc("/api/repos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleReposAPI(w, r, root)
		case http.MethodPost:
			handleAddRepoAPI(w, r, root)
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
		if !validatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		handleUpdateRepoTokenAPI(w, r, repo)
	})
	mux.HandleFunc("/api/repos/{repo}/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !validatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		handleSyncRepoAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/repos/{repo}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !validatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		handleDeleteRepoAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !validatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		handleSessionsAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions/{sessionID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		sid := r.PathValue("sessionID")
		if !validatePathValue(repo) || !validatePathValue(sid) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodDelete {
			handleDeleteSessionAPI(w, r, root, repo, sid)
			return
		}
		handleSessionAPI(w, r, root, repo, sid)
	})
	mux.HandleFunc("/api/repos/{repo}/sessions/{sessionID}/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		sid := r.PathValue("sessionID")
		if !validatePathValue(repo) || !validatePathValue(sid) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		handleGenerateAuditReportAPI(w, r, root, repo, sid)
	})
	mux.HandleFunc("/api/repos/{repo}/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		repo := r.PathValue("repo")
		if !validatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		handleRepoStatusAPI(w, r, root, repo)
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleAllSessionsAPI(w, r, root)
	})
	mux.HandleFunc("/api/reviews", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleCreateReviewTaskAPI(w, r, root)
	})
	mux.HandleFunc("/api/reviews/{taskID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		taskID := r.PathValue("taskID")
		if !validatePathValue(taskID) && !strings.HasPrefix(taskID, "review-") {
			http.Error(w, "invalid task id", http.StatusBadRequest)
			return
		}
		handleReviewTaskAPI(w, r, taskID)
	})
	mux.HandleFunc("/api/config/llm", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleLLMConfigAPI(w, r)
		case http.MethodPost:
			handleSaveLLMConfigAPI(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleRulesAPI(w, r, root)
	})
	mux.HandleFunc("/api/reports", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleListAuditReportsAPI(w, r, root)
	})
	mux.HandleFunc("/api/reports/{reportID}", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reportID := r.PathValue("reportID")
		if !validatePathValue(reportID) {
			http.Error(w, "invalid report id", http.StatusBadRequest)
			return
		}
		handleGetAuditReportAPI(w, r, root, reportID)
	})
	mux.HandleFunc("/api/branches", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleListBranchesAPI(w, r, root)
	})

	// Routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRepos(w, r, root)
	})
	mux.HandleFunc("/r/{repo}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.PathValue("repo")
		if !validatePathValue(repo) {
			http.Error(w, "invalid repo path", http.StatusBadRequest)
			return
		}
		handleSessions(w, r, root, repo)
	})
	mux.HandleFunc("/r/{repo}/{sessionID}", func(w http.ResponseWriter, r *http.Request) {
		repo := r.PathValue("repo")
		sid := r.PathValue("sessionID")
		if !validatePathValue(repo) || !validatePathValue(sid) {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		handleSession(w, r, root, repo, sid)
	})

	// Wrap the mux with a Host-header allowlist. Without this, any web page
	// the user visits can DNS-rebind its origin to 127.0.0.1 and read the
	// session JSONL exposed by this viewer (which contains LLM request bodies
	// = source code being reviewed and the LLM's analysis of it).
	allowed := resolveAllowedHostsFromEnv(addr)
	guarded := hostGuard(allowed, mux)

	srv := &http.Server{
		Addr:    addr,
		Handler: guarded,
	}

	fmt.Printf("\nOpen browser: http://%s\n", addr)
	return srv.ListenAndServe()
}

var cstZone = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	return loc
}()

func formatTime(t time.Time) string {
	return t.In(cstZone).Format("2006-01-02 15:04")
}

func parseTemplate(name string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatDuration": formatDuration,
		"formatTime":     formatTime,
		"truncate":       truncateText,
		"add":            func(a, b int) int { return a + b },
		"cardCount": func(tasks map[TaskType][]*TaskCard) int {
			n := 0
			for _, cards := range tasks {
				n += len(cards)
			}
			return n
		},
		"taskTypeClass": func(tt TaskType) string {
			switch tt {
			case PlanTask:
				return "task-plan"
			case MainTask:
				return "task-main"
			case MemoryCompressionTask:
				return "task-memory"
			case ReLocationTask:
				return "task-relocation"
			default:
				return "task-default"
			}
		},
		"orderedTasks": func(tasks map[TaskType][]*TaskCard) []struct {
			Type  TaskType
			Cards []*TaskCard
		} {
			order := []TaskType{PlanTask, MainTask, ReLocationTask, MemoryCompressionTask}
			var result []struct {
				Type  TaskType
				Cards []*TaskCard
			}
			for _, tt := range order {
				if cards, ok := tasks[tt]; ok {
					result = append(result, struct {
						Type  TaskType
						Cards []*TaskCard
					}{tt, cards})
				}
			}
			for tt, cards := range tasks {
				if tt != PlanTask && tt != MainTask && tt != ReLocationTask && tt != MemoryCompressionTask {
					result = append(result, struct {
						Type  TaskType
						Cards []*TaskCard
					}{tt, cards})
				}
			}
			return result
		},
	}
	content, err := assets.ReadFile("templates/" + name)
	if err != nil {
		return nil, err
	}
	return template.New(name).Funcs(funcMap).Parse(string(content))
}

func truncateText(n int, s string) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func renderTemplate(w http.ResponseWriter, name string, data any) {
	tmpl, err := parseTemplate(name)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		// Partially written — just log
		fmt.Printf("[viewer] template execution error: %v\n", err)
	}
}

func staticFS() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

func formatDuration(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", seconds)
	}
	minutes := int(d.Minutes())
	sec := int(d.Seconds()) - minutes*60
	return fmt.Sprintf("%dm%ds", minutes, sec)
}
