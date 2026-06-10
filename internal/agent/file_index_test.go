package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-code-review/open-code-review/internal/model"
)

func TestMatchHintsUsesPathAndContent(t *testing.T) {
	matches := matchHints("src/services/user.go", "func ValidateToken() {}", []string{"auth", "token", "service"})
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %#v", len(matches), matches)
	}
}

func TestSummarizeFileSample(t *testing.T) {
	summary := summarizeFileSample("internal/server/router.go", `package server

func NewRouter() {}
type Server struct{}
var ignored = true`)
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if !strings.Contains(summary, "func NewRouter") || !strings.Contains(summary, "type Server") {
		t.Fatalf("unexpected summary: %s", summary)
	}
}

func TestBuildLightFileIndexReusesCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repoDir := t.TempDir()
	path := "internal/auth/token.go"
	fullPath := filepath.Join(repoDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("package auth\n\nfunc ValidateToken() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	agent := &Agent{args: Args{RepoDir: repoDir}}
	diffs := []model.Diff{{NewPath: path}}
	hints := []string{"token"}
	first := agent.buildLightFileIndex(diffs, hints)
	second := agent.buildLightFileIndex(diffs, hints)

	if len(first[path].HintMatches) != 1 || len(second[path].HintMatches) != 1 {
		t.Fatalf("expected cached hint match, got first=%#v second=%#v", first[path], second[path])
	}
	if first[path].Summary != second[path].Summary {
		t.Fatalf("expected cached summary to match, first=%q second=%q", first[path].Summary, second[path].Summary)
	}
}

func TestBuildLightFileIndexBuildsSummariesWithoutHints(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repoDir := t.TempDir()
	path := "internal/server/router.go"
	fullPath := filepath.Join(repoDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte("package server\n\nfunc NewRouter() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	agent := &Agent{args: Args{RepoDir: repoDir}}
	index := agent.buildLightFileIndex([]model.Diff{{NewPath: path}}, nil)
	if !strings.Contains(index[path].Summary, "func NewRouter") {
		t.Fatalf("expected summary without hints, got %#v", index[path])
	}
}

func TestRepoFilePathRejectsUnsafePaths(t *testing.T) {
	repoDir := t.TempDir()
	for _, path := range []string{"../secret.go", "/tmp/secret.go", ".."} {
		if fullPath, ok := repoFilePath(repoDir, path); ok {
			t.Fatalf("expected unsafe path %q to be rejected, got %q", path, fullPath)
		}
	}

	fullPath, ok := repoFilePath(repoDir, "internal/agent/file_index.go")
	if !ok || !strings.HasPrefix(fullPath, repoDir) {
		t.Fatalf("expected repo path to be accepted, got ok=%v path=%q", ok, fullPath)
	}
}

func TestBuildLightFileIndexDoesNotReadSymlinkTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repoDir := t.TempDir()
	targetPath := filepath.Join(t.TempDir(), "target.go")
	if err := os.WriteFile(targetPath, []byte("package outside\n\nfunc OutsideSecret() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(repoDir, "internal", "linked.go")
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetPath, symlinkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	agent := &Agent{args: Args{RepoDir: repoDir}}
	index := agent.buildLightFileIndex([]model.Diff{{NewPath: "internal/linked.go", Diff: "+func DiffOnly() {}"}}, []string{"outside"})

	entry := index["internal/linked.go"]
	if strings.Contains(entry.Summary, "OutsideSecret") || len(entry.HintMatches) != 0 {
		t.Fatalf("expected symlink target to be ignored, got %#v", entry)
	}
}

func TestBuildRelatedChangeContextUsesSummariesAndLimitsFiles(t *testing.T) {
	current := model.Diff{NewPath: "internal/auth/service.go", Diff: "+func ValidateToken(token string) error { return nil }"}
	diffs := []model.Diff{current}
	index := lightFileIndex{}
	for _, path := range []string{
		"internal/auth/token.go",
		"internal/auth/session.go",
		"internal/auth/user.go",
		"internal/auth/claims.go",
		"internal/auth/password.go",
		"internal/auth/login.go",
		"internal/auth/middleware.go",
		"internal/auth/store.go",
		"internal/metrics/counter.go",
	} {
		diffs = append(diffs, model.Diff{NewPath: path})
		index[path] = lightFileEntry{Path: path, Summary: "func ValidateToken() {}"}
	}

	agent := &Agent{diffs: diffs, lightIndex: index}
	context := agent.buildRelatedChangeContext(current.NewPath, current.Diff)

	if strings.Count(context, "summary:") != maxRagRelatedFiles {
		t.Fatalf("expected %d related summaries, got context:\n%s", maxRagRelatedFiles, context)
	}
	if !strings.Contains(context, "internal/auth/token.go") || strings.Contains(context, current.NewPath) {
		t.Fatalf("unexpected related context:\n%s", context)
	}
}

func TestPlanTemplateReplacementsIncludesRagContext(t *testing.T) {
	replacements := planTemplateReplacements("today", "rule", "related summaries", "+diff", "background", "tools")
	content := templateReplaceOnce("{{change_files}}\n{{rag_context}}\n{{diff}}", replacements)

	if strings.Contains(content, "{{rag_context}}") || strings.Contains(content, "{{change_files}}") {
		t.Fatalf("expected plan placeholders to be replaced, got %q", content)
	}
	if strings.Count(content, "related summaries") != 2 {
		t.Fatalf("expected related context in both plan placeholders, got %q", content)
	}
}

func TestCompactSummary(t *testing.T) {
	longSummary := strings.Repeat("x", 300)
	got := compactSummary(longSummary)
	if len(got) != 243 || !strings.HasSuffix(got, "...") {
		t.Fatalf("expected compacted summary with ellipsis, got length=%d value=%q", len(got), got)
	}
}
