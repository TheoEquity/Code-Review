package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStripModelSuffix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"claude-opus-4-7[1m]", "claude-opus-4-7"},
		{"claude-sonnet-4-6[2m]", "claude-sonnet-4-6"},
		{"claude-opus-4-7[10m]", "claude-opus-4-7"},
		{"claude-opus-4-7", "claude-opus-4-7"},
		{"", ""},
		{"claude[1m]-extra", "claude[1m]-extra"},
		{"claude-opus-4-7[m]", "claude-opus-4-7[m]"},
		{"claude-opus-4-7[1M]", "claude-opus-4-7[1M]"},
		{"claude-opus-4-7[1]", "claude-opus-4-7[1]"},
	}

	for _, tt := range tests {
		got := stripModelSuffix(tt.input)
		if got != tt.want {
			t.Errorf("stripModelSuffix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveEndpoint_CCEnvStripsModelSuffix(t *testing.T) {
	t.Setenv("OCR_LLM_URL", "")
	t.Setenv("OCR_LLM_TOKEN", "")
	t.Setenv("OCR_LLM_MODEL", "")
	t.Setenv("ANTHROPIC_BASE_URL", "https://api.example.com")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_MODEL", "claude-opus-4-7[1m]")

	ep, err := ResolveEndpoint(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Model != "claude-opus-4-7" {
		t.Errorf("expected model %q, got %q", "claude-opus-4-7", ep.Model)
	}
	if ep.Source != "Claude Code environment" {
		t.Errorf("expected source %q, got %q", "Claude Code environment", ep.Source)
	}
}

func TestResolveEndpoint_CCEnvCleanModelUnchanged(t *testing.T) {
	t.Setenv("OCR_LLM_URL", "")
	t.Setenv("OCR_LLM_TOKEN", "")
	t.Setenv("OCR_LLM_MODEL", "")
	t.Setenv("ANTHROPIC_BASE_URL", "https://api.example.com")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_MODEL", "claude-opus-4-7")

	ep, err := ResolveEndpoint(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Model != "claude-opus-4-7" {
		t.Errorf("expected model %q, got %q", "claude-opus-4-7", ep.Model)
	}
}

func TestResolveEndpoint_OCREnvStripsModelSuffix(t *testing.T) {
	t.Setenv("OCR_LLM_URL", "https://api.example.com/v1/messages")
	t.Setenv("OCR_LLM_TOKEN", "test-token")
	t.Setenv("OCR_LLM_MODEL", "claude-haiku[2m]")

	ep, err := ResolveEndpoint(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Model != "claude-haiku" {
		t.Errorf("expected model %q, got %q", "claude-haiku", ep.Model)
	}
	if ep.Source != "OCR environment" {
		t.Errorf("expected source %q, got %q", "OCR environment", ep.Source)
	}
}

func TestResolveEndpoint_ConfigFileStripsModelSuffix(t *testing.T) {
	t.Setenv("OCR_LLM_URL", "")
	t.Setenv("OCR_LLM_TOKEN", "")
	t.Setenv("OCR_LLM_MODEL", "")
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_MODEL", "")

	cfg := configFile{
		Llm: llmFileConfig{
			URL:       "https://api.example.com/v1/messages",
			AuthToken: "test-token",
			Model:     "gpt-4[1m]",
		},
	}
	data, _ := json.Marshal(cfg)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(cfgPath, data, 0644)

	ep, err := ResolveEndpoint(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Model != "gpt-4" {
		t.Errorf("expected model %q, got %q", "gpt-4", ep.Model)
	}
	if ep.Source != "OCR config file" {
		t.Errorf("expected source %q, got %q", "OCR config file", ep.Source)
	}
}

func TestResolveEndpoints_ConfigFileProviders(t *testing.T) {
	t.Setenv("OCR_LLM_URL", "")
	t.Setenv("OCR_LLM_TOKEN", "")
	t.Setenv("OCR_LLM_MODEL", "")
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_MODEL", "")

	useOpenAI := false
	useAnthropic := true
	cfg := configFile{
		Llm: llmFileConfig{
			Providers: []llmFileConfig{
				{Name: "qwen", URL: "https://dashscope.example.com", AuthToken: "token-a", Model: "qwen-plus[1m]", UseAnthropic: &useAnthropic},
				{Name: "glm", URL: "https://glm.example.com/v1", AuthToken: "token-b", Model: "glm-4", UseAnthropic: &useOpenAI},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	endpoints, err := ResolveEndpoints(cfgPath)
	if err != nil {
		t.Fatalf("ResolveEndpoints: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints[0].Name != "qwen" || endpoints[0].Model != "qwen-plus" || endpoints[0].Protocol != "anthropic" {
		t.Fatalf("unexpected first endpoint: %#v", endpoints[0])
	}
	if endpoints[1].Name != "glm" || endpoints[1].Model != "glm-4" || endpoints[1].Protocol != "openai" {
		t.Fatalf("unexpected second endpoint: %#v", endpoints[1])
	}
}

func TestResolveEndpoints_ProvidersOverrideTopLevel(t *testing.T) {
	t.Setenv("OCR_LLM_URL", "")
	t.Setenv("OCR_LLM_TOKEN", "")
	t.Setenv("OCR_LLM_MODEL", "")
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_MODEL", "")

	useOpenAI := false
	cfg := configFile{
		Llm: llmFileConfig{
			Name:      "top-level",
			URL:       "https://top.example.com",
			AuthToken: "top-token",
			Model:     "top-model",
			Providers: []llmFileConfig{
				{Name: "provider", URL: "https://provider.example.com/v1", AuthToken: "provider-token", Model: "provider-model", UseAnthropic: &useOpenAI},
			},
		},
	}
	data, _ := json.Marshal(cfg)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	endpoints, err := ResolveEndpoints(cfgPath)
	if err != nil {
		t.Fatalf("ResolveEndpoints: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected providers to override top-level endpoint, got %d endpoints", len(endpoints))
	}
	if endpoints[0].Name != "provider" || endpoints[0].Model != "provider-model" {
		t.Fatalf("unexpected endpoint: %#v", endpoints[0])
	}
}
