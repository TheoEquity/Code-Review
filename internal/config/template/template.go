// Package template loads and validates task prompt templates for the code review agent.
package template

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// Template holds the native agent task template configuration.
// Mirrors NativeAgentTemplate from the Java implementation, loaded via JSON at runtime.
type Template struct {
	MainTask              LlmConversation  `json:"MAIN_TASK"`
	PlanTask              *LlmConversation `json:"PLAN_TASK,omitempty"`
	MemoryCompressionTask LlmConversation  `json:"MEMORY_COMPRESSION_TASK"`
	MaxTokens             int              `json:"MAX_TOKENS"`
	ToolRequestWaitTimeMs int              `json:"TOOL_REQUEST_WAIT_TIME_MS"`
	MaxToolRequestTimes   int              `json:"MAX_TOOL_REQUEST_TIMES"`
	MaxSubtaskExecMinutes int              `json:"MAX_SUBTASK_EXECUTION_TIME_MINUTES"`
	PlanModeLineThreshold int              `json:"PLAN_MODE_LINE_THRESHOLD"`
	ReLocationTask        *LlmConversation `json:"RE_LOCATION_TASK,omitempty"`
}

//go:embed task_template.json
var defaultTemplate []byte

// LoadDefault parses the embedded task_template.json.
func LoadDefault() (*Template, error) {
	var tpl Template
	if err := json.Unmarshal(defaultTemplate, &tpl); err != nil {
		return nil, fmt.Errorf("unmarshal default template: %w", err)
	}
	tpl.slimDefaultPrompts()
	return &tpl, nil
}

func (t *Template) slimDefaultPrompts() {
	setFirstSystem(&t.MainTask, `You are a concise code review assistant.

Rules:
- Review only newly added or modified code in the provided diff.
- Report real correctness, security, performance, reliability, or maintainability issues.
- Use tools when required context is missing.
- Avoid comments on deleted, unchanged, generated, formatting-only, or correct code.
- Output actionable review findings only.`)

	setFirstUser(&t.MainTask, `// Related changed files selected by lightweight RAG.
<related_change_context>
{{rag_context}}
</related_change_context>

<current_file_path>{{current_file_path}}</current_file_path>

<current_file_diff>
{{diff}}
</current_file_diff>

Current time in the real world: {{current_system_date_time}}

<user_task>
### Requirement Background (Optional)
{{requirement_background}}

### Review Checklist
{{system_rule}}

### Review Plan (Optional)
{{plan_guidance}}

Review the code changes in <current_file_diff>.
</user_task>`)

	if t.PlanTask != nil {
		setFirstSystem(t.PlanTask, `You are a concise code review planning assistant.

Return only the required JSON review plan.`)
		setFirstUser(t.PlanTask, `// Related changed files selected by lightweight RAG.
<related_change_context>
{{rag_context}}
</related_change_context>

<current_file_path>{{current_file_path}}</current_file_path>

<current_file_diff>
{{diff}}
</current_file_diff>

Current time in the real world: {{current_system_date_time}}

### Requirement Background (Optional)
{{requirement_background}}

### Review Checklist
{{system_rule}}

### Available Tools
{{plan_tools}}

### JSON Schema
{
  "change_summary": "A brief description of the purpose and scope of this code change",
  "issues": [
    {
      "severity": "high|medium|low",
      "description": "Problem location, nature, and impact",
      "tool_guidance": [
        {
          "name": "Tool name",
          "reason": "Why this tool is relevant",
          "arguments": "Invocation arguments"
        }
      ]
    }
  ]
}

### Rules
Only analyze newly added and modified code. Sort issues by severity. Start with `+"```json"+`.`)
	}

	setFirstSystem(&t.MemoryCompressionTask, `Compress the review conversation into concise state for continuation. Include confirmed issues, useful tool conclusions, completed work, pending work, and current focus when present.`)
	if t.ReLocationTask != nil {
		setFirstSystem(t.ReLocationTask, `Extract the exact code snippet from the diff that the review comment targets. Output only one fenced code block. /no_think`)
	}
}

func setFirstSystem(conv *LlmConversation, content string) {
	for i := range conv.Messages {
		if conv.Messages[i].Role == "system" {
			conv.Messages[i].Content = content
			return
		}
	}
}

func setFirstUser(conv *LlmConversation, content string) {
	for i := range conv.Messages {
		if conv.Messages[i].Role == "user" {
			conv.Messages[i].Content = content
			return
		}
	}
}

// applyLanguage appends instruction to all system-role messages in conv.
func applyLanguage(conv *LlmConversation, instruction string) {
	for i := range conv.Messages {
		if conv.Messages[i].Role == "system" {
			conv.Messages[i].Content += instruction
		}
	}
}

// resolveLang returns the resolved language name for the instruction.
func resolveLang(lang string) string {
	if lang == "" {
		return "Chinese"
	}
	return lang
}

// ApplyLanguage injects a language directive into all system-role messages
// across MAIN_TASK, PLAN_TASK (if set), and MEMORY_COMPRESSION_TASK.
func (t *Template) ApplyLanguage(lang string) {
	instruction := "\n\nAlways respond in " + resolveLang(lang) + "."
	applyLanguage(&t.MainTask, instruction)
	if t.PlanTask != nil {
		applyLanguage(t.PlanTask, instruction)
	}
	applyLanguage(&t.MemoryCompressionTask, instruction)
}
func (t *Template) Validate() error {
	if t.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	if t.MaxToolRequestTimes <= 0 {
		return fmt.Errorf("max_tool_request_times must be positive")
	}
	if len(t.MainTask.Messages) == 0 {
		return fmt.Errorf("main_task.messages must not be empty")
	}
	return nil
}

// LlmConversation mirrors LlmConversation from the Java side — a preset prompt with settings.
type LlmConversation struct {
	Timeout  int           `json:"timeout"`
	Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
