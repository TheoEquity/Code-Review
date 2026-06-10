package viewer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleSaveLLMConfigAPI_DoesNotCreateProviderCycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	body := []byte(`{
		"providers": [
			{"name":"primary","url":"https://primary.example.com/v1","authToken":"token-1","model":"model-1","useAnthropic":false,"extraBody":"{\"temperature\":0.2}"},
			{"name":"fallback-1","url":"https://fallback.example.com/v1","authToken":"token-2","model":"model-2","useAnthropic":false}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/llm", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	handleSaveLLMConfigAPI(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(home, ".opencodereview", "config.json"))
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	var saved map[string]any
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("saved config is invalid json: %v\n%s", err, string(data))
	}
	llmSection, ok := saved["llm"].(map[string]any)
	if !ok {
		t.Fatalf("missing llm section: %#v", saved)
	}
	providers, ok := llmSection["providers"].([]any)
	if !ok || len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %#v", llmSection["providers"])
	}
	firstProvider, ok := providers[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first provider: %#v", providers[0])
	}
	if _, exists := firstProvider["providers"]; exists {
		t.Fatalf("first provider should not contain nested providers: %#v", firstProvider)
	}
}

func TestHandleLLMConfigAPI_RedactsAuthToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".opencodereview")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := []byte(`{"llm":{"url":"https://primary.example.com/v1","auth_token":"secret-token","model":"model-1","use_anthropic":false,"providers":[{"name":"primary","url":"https://primary.example.com/v1","auth_token":"secret-token","model":"model-1","use_anthropic":false}]}}`)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), config, 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config/llm", nil)
	recorder := httptest.NewRecorder()

	handleLLMConfigAPI(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("secret-token")) {
		t.Fatalf("response leaked auth token: %s", recorder.Body.String())
	}
	var response llmConfigResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if response.Config.AuthToken != "" || len(response.Providers) != 1 || response.Providers[0].AuthToken != "" {
		t.Fatalf("expected redacted tokens, got %#v", response)
	}
	if !response.Config.HasAuthToken || !response.Providers[0].HasAuthToken {
		t.Fatalf("expected hasAuthToken flags, got %#v", response)
	}
}

func TestHandleSaveLLMConfigAPI_EmptyAuthTokenPreservesExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".opencodereview")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.json")
	config := []byte(`{"llm":{"url":"https://primary.example.com/v1","auth_token":"secret-token","model":"model-1","use_anthropic":false,"providers":[{"name":"primary","url":"https://primary.example.com/v1","auth_token":"secret-token","model":"model-1","use_anthropic":false}]}}`)
	if err := os.WriteFile(configPath, config, 0o644); err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"providers":[{"name":"primary","url":"https://primary.example.com/v1","authToken":"","model":"model-1","useAnthropic":false}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config/llm", bytes.NewReader(body))
	recorder := httptest.NewRecorder()

	handleSaveLLMConfigAPI(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !bytes.Contains(data, []byte("secret-token")) {
		t.Fatalf("expected saved config to preserve existing token: %s", string(data))
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte("secret-token")) {
		t.Fatalf("save response leaked auth token: %s", recorder.Body.String())
	}
}
