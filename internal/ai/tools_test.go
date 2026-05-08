package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHistoryToMessagesInitialPromptOnly(t *testing.T) {
	msgs, err := historyToMessages(History{UserPrompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestHistoryToMessagesRequiresPrompt(t *testing.T) {
	_, err := historyToMessages(History{})
	if err == nil || !strings.Contains(err.Error(), "UserPrompt") {
		t.Fatalf("expected UserPrompt error, got %v", err)
	}
}

func TestHistoryToMessagesAssistantAndUserToolResult(t *testing.T) {
	hist := History{
		UserPrompt: "what about loyalty?",
		Turns: []HistoryTurn{
			{
				Role: "assistant",
				Text: "let me check",
				ToolUses: []ToolUse{{
					ID:    "tu_1",
					Name:  "grep",
					Input: json.RawMessage(`{"query":"loyalty"}`),
				}},
			},
			{
				Role: "user",
				ToolResults: []ToolResult{{
					ToolUseID: "tu_1",
					Content:   "matches in episode X",
					IsError:   false,
				}},
			},
		},
	}
	msgs, err := historyToMessages(hist)
	if err != nil {
		t.Fatal(err)
	}
	// Initial user + assistant + user(tool_results) = 3 messages.
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestHistoryToMessagesAssistantWithoutContentRejected(t *testing.T) {
	_, err := historyToMessages(History{
		UserPrompt: "q",
		Turns:      []HistoryTurn{{Role: "assistant"}},
	})
	if err == nil {
		t.Fatal("expected error for empty assistant turn")
	}
}

func TestHistoryToMessagesUserWithoutResultsRejected(t *testing.T) {
	_, err := historyToMessages(History{
		UserPrompt: "q",
		Turns:      []HistoryTurn{{Role: "user"}},
	})
	if err == nil {
		t.Fatal("expected error for user turn without tool_results")
	}
}

func TestHistoryToMessagesUnknownRole(t *testing.T) {
	_, err := historyToMessages(History{
		UserPrompt: "q",
		Turns:      []HistoryTurn{{Role: "system"}},
	})
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
}

func TestHistoryToMessagesBadToolUseInput(t *testing.T) {
	_, err := historyToMessages(History{
		UserPrompt: "q",
		Turns: []HistoryTurn{{
			Role: "assistant",
			ToolUses: []ToolUse{{
				ID:    "tu_1",
				Name:  "grep",
				Input: json.RawMessage("not json"),
			}},
		}},
	})
	if err == nil {
		t.Fatal("expected JSON decode error on invalid tool input")
	}
}

func TestToolDefsToSDKTranslation(t *testing.T) {
	defs := []ToolDef{
		{
			Name:        "grep",
			Description: "search for stuff",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {"query": {"type": "string"}},
				"required": ["query"]
			}`),
		},
		{
			Name: "no_desc",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
	}
	got, err := toolDefsToSDK(defs)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tool params, got %d", len(got))
	}
	if got[0].OfTool == nil {
		t.Fatal("expected OfTool to be set")
	}
	if got[0].OfTool.Name != "grep" {
		t.Errorf("name: %q", got[0].OfTool.Name)
	}
	if !got[0].OfTool.Description.Valid() {
		t.Error("description should be set")
	}
	if got[0].OfTool.Description.Value != "search for stuff" {
		t.Errorf("description: %q", got[0].OfTool.Description.Value)
	}
	if len(got[0].OfTool.InputSchema.Required) != 1 || got[0].OfTool.InputSchema.Required[0] != "query" {
		t.Errorf("required: %v", got[0].OfTool.InputSchema.Required)
	}
	// Tool with no description: Valid() should be false.
	if got[1].OfTool.Description.Valid() {
		t.Error("expected no description")
	}
}

func TestToolDefsToSDKBadSchema(t *testing.T) {
	_, err := toolDefsToSDK([]ToolDef{{
		Name:        "broken",
		InputSchema: json.RawMessage("not json"),
	}})
	if err == nil {
		t.Fatal("expected schema decode error")
	}
}

func TestToolDefsToSDKEmpty(t *testing.T) {
	got, err := toolDefsToSDK(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}
