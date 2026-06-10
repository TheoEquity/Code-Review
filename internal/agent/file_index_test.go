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
