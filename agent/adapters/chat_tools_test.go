package adapters

import (
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

func TestToolsToChatTools_InlineOfFunctionProducesChatTools(t *testing.T) {
	tools := []responses.ToolUnionParam{{
		OfFunction: &responses.FunctionToolParam{
			Name:        "todo_set",
			Description: openai.String("track todos"),
			Parameters: map[string]any{
				"type": "object",
			},
			Strict: openai.Bool(true),
		},
	}}
	out := toolsToChatTools(tools)
	if len(out) != 1 {
		t.Fatalf("expected 1 chat tool, got %d", len(out))
	}
	if out[0].OfFunction == nil || out[0].OfFunction.Function.Name != "todo_set" {
		t.Fatalf("unexpected tool: %#v", out[0].OfFunction)
	}
}

func TestResponsesToChatMessages_IncludesFunctionCallAsAssistantToolCalls(t *testing.T) {
	items := []responses.ResponseInputItemUnionParam{
		responses.ResponseInputItemParamOfFunctionCall(`{"x":1}`, "call_function_mp6oynxmen1a_1", "my_tool"),
		responses.ResponseInputItemParamOfFunctionCallOutput("call_function_mp6oynxmen1a_1", "ok"),
	}
	msgs, err := responsesToChatMessages(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 chat messages, got %d", len(msgs))
	}
	if msgs[0].OfAssistant == nil || len(msgs[0].OfAssistant.ToolCalls) != 1 {
		t.Fatalf("first message should be assistant with 1 tool_call, got %#v", msgs[0])
	}
	if id := *msgs[0].OfAssistant.ToolCalls[0].GetID(); id != "call_function_mp6oynxmen1a_1" {
		t.Fatalf("tool call id: %q", id)
	}
	if msgs[1].OfTool == nil || msgs[1].OfTool.ToolCallID != "call_function_mp6oynxmen1a_1" {
		t.Fatalf("tool message id mismatch: %#v", msgs[1].OfTool)
	}
}
