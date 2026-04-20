package adapters

import (
	"context"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// responsesAdapter implements LLMClient using the OpenAI Responses API (/v1/responses).
// Use this for models that support the Responses API natively.
type responsesAdapter struct {
	client openai.Client
}

func newResponsesAdapter(client openai.Client) LLMClient {
	return &responsesAdapter{client: client}
}

func (a *responsesAdapter) AdapterName() string { return "responses-api" }

func (a *responsesAdapter) Chat(ctx context.Context, model string,
	tools []responses.ToolUnionParam,
	messages []responses.ResponseInputItemUnionParam) (string, []responses.ResponseInputItemUnionParam, error) {

	ctx2, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	resp, err := a.client.Responses.New(ctx2, responses.ResponseNewParams{
		Model: openai.ResponsesModel(model),
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: messages},
		Tools: tools,
	})
	if err != nil {
		return "", nil, err
	}

	var text string
	var toolCalls []responses.ResponseInputItemUnionParam

	for _, item := range resp.Output {
		if item.Type == "message" {
			if item.Content != nil {
				for _, block := range item.Content {
					if block.Type == "output_text" && block.Text != "" {
						text = block.Text
					}
				}
			}
		}
		if item.Type == "function_call" {
			toolCalls = append(toolCalls, responses.ResponseInputItemParamOfFunctionCall(
				item.Arguments, item.CallID, item.Name,
			))
		}
	}
	return strings.TrimSpace(text), toolCalls, nil
}

func (a *responsesAdapter) SimpleChat(ctx context.Context, model string,
	messages []responses.ResponseInputItemUnionParam) (string, error) {

	ctx2, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := a.client.Responses.New(ctx2, responses.ResponseNewParams{
		Model: openai.ResponsesModel(model),
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: messages},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.OutputText()), nil
}
