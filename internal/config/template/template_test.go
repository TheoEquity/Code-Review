package template

import (
	"strings"
	"testing"
)

func TestLoadDefaultSlimsSystemPrompts(t *testing.T) {
	tpl, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}

	mainSystem := firstRoleContent(tpl.MainTask, "system")
	if len(mainSystem) > 500 {
		t.Fatalf("expected slim main system prompt, got %d chars", len(mainSystem))
	}
	if strings.Contains(mainSystem, "developed by Alibaba") {
		t.Fatalf("expected legacy verbose system prompt to be replaced: %s", mainSystem)
	}

	mainUser := firstRoleContent(tpl.MainTask, "user")
	if !strings.Contains(mainUser, "{{rag_context}}") || !strings.Contains(mainUser, "{{diff}}") || !strings.Contains(mainUser, "{{system_rule}}") {
		t.Fatalf("main user prompt lost required placeholders: %s", mainUser)
	}
}

func TestLoadDefaultKeepsPlanToolsInUserPrompt(t *testing.T) {
	tpl, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	if tpl.PlanTask == nil {
		t.Fatal("expected default plan task")
	}

	planSystem := firstRoleContent(*tpl.PlanTask, "system")
	if strings.Contains(planSystem, "{{plan_tools}}") {
		t.Fatalf("expected plan tools outside system prompt: %s", planSystem)
	}

	planUser := firstRoleContent(*tpl.PlanTask, "user")
	if !strings.Contains(planUser, "{{plan_tools}}") || !strings.Contains(planUser, "{{rag_context}}") {
		t.Fatalf("plan user prompt lost required placeholders: %s", planUser)
	}
}

func firstRoleContent(conv LlmConversation, role string) string {
	for _, msg := range conv.Messages {
		if msg.Role == role {
			return msg.Content
		}
	}
	return ""
}
