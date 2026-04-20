package adapters

import (
	"context"
	"log"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// LLMClient is the single interface that all LLM adapters implement.
// Runtime and server hold this interface — they never call a specific API directly.
type LLMClient interface {
	// Chat performs a tool-use chat turn: returns text response and any function calls.
	Chat(ctx context.Context, model string, tools []responses.ToolUnionParam,
		messages []responses.ResponseInputItemUnionParam) (text string, toolCalls []responses.ResponseInputItemUnionParam, err error)

	// SimpleChat is a single-turn chat without tools (used for summarization/compaction).
	SimpleChat(ctx context.Context, model string,
		messages []responses.ResponseInputItemUnionParam) (text string, err error)

	// AdapterName returns a short identifier for logging/debugging.
	AdapterName() string
}

// NewLLMClient creates the appropriate LLMClient based on the model or base URL.
// MiniMax-family models use Chat Completions API; everything else uses Responses API.
func NewLLMClient(client openai.Client, baseURL, model string) LLMClient {
	if isChatCompletionsModel(model, baseURL) {
		log.Printf("[adapter] using Chat Completions adapter (model=%s, base=%s)", model, baseURL)
		return newChatAdapter(client)
	}
	log.Printf("[adapter] using Responses API adapter (model=%s, base=%s)", model, baseURL)
	return newResponsesAdapter(client)
}

// isChatCompletionsModel returns true when the model must use /v1/chat/completions.
// MiniMax does not support /v1/responses.
func isChatCompletionsModel(model, baseURL string) bool {
	model = strings.ToLower(model)
	baseURL = strings.ToLower(baseURL)
	return strings.Contains(baseURL, "minimaxi") ||
		strings.Contains(baseURL, "minimax") ||
		strings.HasPrefix(model, "minimax")
}
