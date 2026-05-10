package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"
)

// chatAdapter implements LLMClient using the Chat Completions API (/v1/chat/completions).
// Use this for models that do NOT support the Responses API, e.g. MiniMax.
type chatAdapter struct {
	client openai.Client
}

func envTimeoutSeconds(key string, fallback int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(v) * time.Second
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

	// Larger timeout for tool-heavy turns (e.g. ppt generation reading multiple files).
	ctx2, cancel := context.WithTimeout(ctx, envTimeoutSeconds("CHAT_COMPLETION_TIMEOUT_SEC", 180))
	defer cancel()

	resp, err := a.chatCompletion(ctx2, model, chatMsgs, chatTools, maxTokens)
	if err != nil {
		return "", nil, err
	}
	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	msg := choice.Message
	text := msg.Content

	toolCalls, toolErr := toResponseToolCalls(msg.ToolCalls)
	if toolErr != nil && choice.FinishReason == "length" {
		// Recovery branch: ask model to re-emit ONLY valid function calls.
		recoveredMsg, recErr := a.recoverFromLength(ctx2, model, chatMsgs, chatTools, maxTokens, msg)
		if recErr != nil {
			return "", nil, fmt.Errorf("chat completion truncated and recovery failed: %w", recErr)
		}
		if strings.TrimSpace(text) == "" {
			text = recoveredMsg.Content
		}
		toolCalls, toolErr = toResponseToolCalls(recoveredMsg.ToolCalls)
	}
	if toolErr != nil {
		return "", nil, toolErr
	}
	if choice.FinishReason == "length" && len(toolCalls) == 0 && strings.TrimSpace(text) == "" {
		return "", nil, fmt.Errorf("chat completion truncated (finish_reason=length): no usable content, please retry")
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

	ctx2, cancel := context.WithTimeout(ctx, envTimeoutSeconds("CHAT_SIMPLE_TIMEOUT_SEC", 120))
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
		case responses.EasyInputMessageRoleDeveloper, responses.EasyInputMessageRoleSystem:
			// Map developer/system→user for MiniMax Chat Completions API.
			// MiniMax does not support the system role in /v1/chat/completions (error 2013).
			// Prefix content so the model still knows this is a system message.
			prefixed := "[System] " + content
			return openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.Opt[string]{Value: prefixed},
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
		case "developer", "system":
			// Map developer/system→user for MiniMax Chat Completions API.
			// MiniMax does not support the system role in /v1/chat/completions (error 2013).
			// Prefix content so the model still knows this is a system message.
			prefixed := "[System] " + content
			return openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: param.Opt[string]{Value: prefixed},
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

	// OfFunctionCall path — must become assistant.tool_calls so the next tool message can reference tool_call_id.
	// Dropping this caused MiniMax 400: "tool result's tool id (...) not found (2013)".
	if item.OfFunctionCall != nil {
		fc := item.OfFunctionCall
		tc := openai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
				ID:   fc.CallID,
				Type: constant.Function("function"),
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      fc.Name,
					Arguments: fc.Arguments,
				},
			},
		}
		return openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Role:      constant.Assistant("assistant"),
				ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{tc},
			},
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
//
// Important: responses.ToolUnionParam marshals OfFunction with json ",inline" — there is
// no top-level "function" key. Do not round-trip through JSON expecting m["function"].
func toolsToChatTools(tools []responses.ToolUnionParam) []openai.ChatCompletionToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		if t.OfFunction == nil {
			continue
		}
		fn := t.OfFunction
		paramsMap := fn.Parameters
		if paramsMap == nil {
			paramsMap = map[string]any{}
		}
		result = append(result, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        fn.Name,
			Description: fn.Description,
			Strict:      fn.Strict,
			Parameters:  shared.FunctionParameters(paramsMap),
		}))
	}
	return result
}

func must(s string, _ error) string { return s }

func (a *chatAdapter) chatCompletion(
	ctx context.Context,
	model string,
	messages []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolUnionParam,
	maxTokens param.Opt[int64],
) (*openai.ChatCompletion, error) {
	if len(tools) > 0 {
		return a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:      model,
			Messages:   messages,
			Tools:      tools,
			ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{OfAuto: param.Opt[string]{Value: "auto"}},
			MaxTokens:  maxTokens,
		})
	}
	return a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     model,
		Messages:  messages,
		MaxTokens: maxTokens,
	})
}

func toResponseToolCalls(toolCalls []openai.ChatCompletionMessageToolCallUnion) ([]responses.ResponseInputItemUnionParam, error) {
	out := make([]responses.ResponseInputItemUnionParam, 0, len(toolCalls))
	for _, tc := range toolCalls {
		argStr, err := normalizeToolArguments(tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			return nil, err
		}
		out = append(out, responses.ResponseInputItemParamOfFunctionCall(
			argStr, tc.ID, tc.Function.Name,
		))
	}
	return out, nil
}

func (a *chatAdapter) recoverFromLength(
	ctx context.Context,
	model string,
	baseMessages []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolUnionParam,
	maxTokens param.Opt[int64],
	lastMsg openai.ChatCompletionMessage,
) (openai.ChatCompletionMessage, error) {
	const repairHint = "[System] Previous assistant output was truncated by max_tokens. Re-emit ONLY complete function tool calls as valid JSON arguments. If writing a large file, use chunked writes: first write_file with initial chunk, then append_file for remaining chunks, and finish with read_file to verify integrity. No prose."
	recoverMsgs := append([]openai.ChatCompletionMessageParamUnion{}, baseMessages...)
	if strings.TrimSpace(lastMsg.Content) != "" {
		recoverMsgs = append(recoverMsgs, openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: param.Opt[string]{Value: lastMsg.Content},
				},
			},
		})
	}
	recoverMsgs = append(recoverMsgs, openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: param.Opt[string]{Value: repairHint},
			},
		},
	})
	resp, err := a.chatCompletion(ctx, model, recoverMsgs, tools, maxTokens)
	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}
	if len(resp.Choices) == 0 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("no choices in recovery response")
	}
	return resp.Choices[0].Message, nil
}

func normalizeToolArguments(toolName, raw string) (string, error) {
	args := strings.TrimSpace(raw)
	if args == "" {
		return "{}", nil
	}

	var payload any
	if err := json.Unmarshal([]byte(args), &payload); err != nil {
		return "", fmt.Errorf("invalid tool args for %s: %w", toolName, err)
	}
	obj, ok := payload.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid tool args for %s: expected JSON object", toolName)
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("invalid tool args for %s: %w", toolName, err)
	}
	return string(out), nil
}
