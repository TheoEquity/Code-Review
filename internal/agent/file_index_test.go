package agent

import (
	"strings"
	"testing"
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
