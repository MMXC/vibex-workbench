package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

// chatAdapter implements LLMClient using the Chat Completions API (/v1/chat/completions).
// Use this for models that do NOT support the Responses API, e.g. MiniMax.
type chatAdapter struct {
	client openai.Client
}

func newChatAdapter(client openai.Client) LLMClient {
	return &chatAdapter{client: client}
}

func (a *chatAdapter) AdapterName() string { return "chat-completions" }

// ── Public Interface ──────────────────────────────────────────────

func (a *chatAdapter) Chat(ctx context.Context, model string,
	tools []responses.ToolUnionParam,
	messages []responses.ResponseInputItemUnionParam) (string, []responses.ResponseInputItemUnionParam, error) {

	chatMsgs, err := responsesToChatMessages(messages)
	if err != nil {
		return "", nil, fmt.Errorf("convert messages: %w", err)
	}
	if len(chatMsgs) == 0 {
		chatMsgs = []openai.ChatCompletionMessageParamUnion{
			openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.Opt[string]{Value: "ping"},
					},
				},
			},
		}
	}

	chatTools := toolsToChatTools(tools)
	maxTokens := param.Opt[int64]{Value: 8192}

	ctx2, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	var resp *openai.ChatCompletion
	if len(chatTools) > 0 {
		resp, err = a.client.Chat.Completions.New(ctx2, openai.ChatCompletionNewParams{
			Model:     model,
			Messages:  chatMsgs,
			Tools:     chatTools,
			MaxTokens: maxTokens,
		})
	} else {
		resp, err = a.client.Chat.Completions.New(ctx2, openai.ChatCompletionNewParams{
			Model:     model,
			Messages:  chatMsgs,
			MaxTokens: maxTokens,
		})
	}
	if err != nil {
		return "", nil, err
	}
	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	msg := resp.Choices[0].Message
	text := msg.Content

	toolCalls := make([]responses.ResponseInputItemUnionParam, 0)
	for _, tc := range msg.ToolCalls {
		args := tc.Function.Arguments
		if args == "" {
			args = "{}"
		}
		var argsMap map[string]interface{}
		json.Unmarshal([]byte(args), &argsMap)
		argStr, _ := json.Marshal(argsMap)
		toolCalls = append(toolCalls, responses.ResponseInputItemParamOfFunctionCall(
			string(argStr), tc.ID, tc.Function.Name,
		))
	}
	return text, toolCalls, nil
}

func (a *chatAdapter) SimpleChat(ctx context.Context, model string,
	messages []responses.ResponseInputItemUnionParam) (string, error) {

	chatMsgs, err := responsesToChatMessages(messages)
	if err != nil {
		return "", fmt.Errorf("convert messages: %w", err)
	}
	if len(chatMsgs) == 0 {
		return "", nil
	}

	ctx2, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	maxTokens := param.Opt[int64]{Value: 8192}
	resp, err := a.client.Chat.Completions.New(ctx2, openai.ChatCompletionNewParams{
		Model:     model,
		Messages:  chatMsgs,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}

// ── Internal Converters ───────────────────────────────────────────

// responsesToChatMessages converts Responses API message items to Chat API message params.
func responsesToChatMessages(items []responses.ResponseInputItemUnionParam) ([]openai.ChatCompletionMessageParamUnion, error) {
	result := make([]openai.ChatCompletionMessageParamUnion, 0, len(items))
	for _, item := range items {
		msg := responseItemToChat(item)
		if isEmptyChatMsg(msg) {
			continue
		}
		result = append(result, msg)
	}
	return result, nil
}

func isEmptyChatMsg(m openai.ChatCompletionMessageParamUnion) bool {
	return m.OfDeveloper == nil && m.OfSystem == nil &&
		m.OfUser == nil && m.OfAssistant == nil && m.OfTool == nil
}

func responseItemToChat(item responses.ResponseInputItemUnionParam) openai.ChatCompletionMessageParamUnion {
	// OfMessage path
	if item.OfMessage != nil {
		msg := item.OfMessage
		content := extractResponseMessageContent(msg.Content)
		role := msg.Role
		switch role {
		case responses.EasyInputMessageRoleDeveloper:
			return openai.ChatCompletionMessageParamUnion{
				OfDeveloper: &openai.ChatCompletionDeveloperMessageParam{
					Content: openai.ChatCompletionDeveloperMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		case responses.EasyInputMessageRoleSystem:
			return openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		case responses.EasyInputMessageRoleAssistant:
			return openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		default: // user
			return openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		}
	}

	// OfInputMessage path (used by tool loop follow-up items)
	if item.OfInputMessage != nil {
		msg := item.OfInputMessage
		content := extractInputMessageContent(msg.Content)
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "developer":
			return openai.ChatCompletionMessageParamUnion{
				OfDeveloper: &openai.ChatCompletionDeveloperMessageParam{
					Content: openai.ChatCompletionDeveloperMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		case "system":
			return openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		case "assistant":
			return openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		default:
			return openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.Opt[string]{Value: content},
					},
				},
			}
		}
	}

	// OfFunctionCallOutput path
	if item.OfFunctionCallOutput != nil {
		callID := strings.TrimSpace(item.OfFunctionCallOutput.CallID)
		output := extractUnionString(item.OfFunctionCallOutput.Output)
		return openai.ChatCompletionMessageParamUnion{
			OfTool: &openai.ChatCompletionToolMessageParam{
				ToolCallID: callID,
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfString: param.Opt[string]{Value: output},
				},
			},
		}
	}

	return openai.ChatCompletionMessageParamUnion{}
}

// extractResponseMessageContent extracts a plain string from EasyInputMessageContentUnionParam.
func extractResponseMessageContent(c responses.EasyInputMessageContentUnionParam) string {
	if c.OfString.Valid() {
		return c.OfString.Value
	}
	return ""
}

// extractInputMessageContent extracts content from ResponseInputMessageContentListParam.
// The type serializes as a plain string in JSON.
func extractInputMessageContent(c responses.ResponseInputMessageContentListParam) string {
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	if text, ok := m["text"].(string); ok {
		return text
	}
	// Fallback: try as raw string
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		return raw
	}
	return string(data)
}

// extractUnionString extracts a string from ResponseInputItemFunctionCallOutputOutputUnionParam.
// Uses JSON serialization as a safe bridge for union types.
func extractUnionString(u responses.ResponseInputItemFunctionCallOutputOutputUnionParam) string {
	data, err := json.Marshal(u)
	if err != nil {
		return ""
	}
	// Try extracting "text" field from the serialized union
	var wrapper struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Text != "" {
		return wrapper.Text
	}
	// Fallback: try as raw string
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		return raw
	}
	return strings.TrimPrefix(strings.TrimPrefix(string(data), `"`), `"`)
}

// toolsToChatTools converts Responses API tool definitions to Chat API tool definitions.
func toolsToChatTools(tools []responses.ToolUnionParam) []openai.ChatCompletionToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		fn := extractToolFunction(t)
		if fn == nil {
			continue
		}
		name := ""
		if n, ok := (*fn)["name"].(string); ok {
			name = n
		}
		desc := ""
		if d, ok := (*fn)["description"].(string); ok {
			desc = d
		}
		var paramsMap map[string]any
		if p, ok := (*fn)["parameters"]; ok {
			data, _ := json.Marshal(p)
			json.Unmarshal(data, &paramsMap)
		}
		if paramsMap == nil {
			paramsMap = map[string]any{}
		}
		result = append(result, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Type: "function",
				Function: openai.FunctionDefinitionParam{
					Name:        name,
					Description: param.Opt[string]{Value: desc},
					Parameters:  shared.FunctionParameters(paramsMap),
				},
			},
		})
	}
	return result
}

// extractToolFunction extracts a function definition map from a ToolUnionParam via JSON.
func extractToolFunction(t responses.ToolUnionParam) *map[string]interface{} {
	data, err := json.Marshal(t)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	fn, ok := m["function"].(map[string]interface{})
	if !ok {
		return nil
	}
	return &fn
}

func must(s string, _ error) string { return s }
