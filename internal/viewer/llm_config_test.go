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
