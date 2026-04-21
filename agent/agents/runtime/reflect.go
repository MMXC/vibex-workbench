package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v3/responses"
	"vibex/agent/adapters"
)

// ReflectionResult captures what self-reflection found and did.
type ReflectionResult struct {
	PatternsFound []string   `json:"patterns_found"`
	ActionsTaken  []string   `json:"actions_taken"`
	Errors        []string   `json:"errors,omitempty"`
}

// reflectAfterTurn runs after each agent turn.
// It inspects the tool calls from this turn, detects automatable patterns,
// and executes improvements directly without prompting the user.
func reflectAfterTurn(
	ctx context.Context,
	llm adapters.LLMClient,
	model string,
	inputItems []responses.ResponseInputItemUnionParam,
	answer string,
) ReflectionResult {
	result := ReflectionResult{}

	// 1. Extract this turn's tool calls from inputItems.
	toolCalls := extractToolCalls(inputItems)

	// 2. Pattern detectors — each returns actions or nil.
	result.PatternsFound, result.ActionsTaken, result.Errors =
		detectAndFix(ctx, llm, model, toolCalls, answer)

	return result
}

// detectAndFix runs all pattern detectors and collects results.
func detectAndFix(
	ctx context.Context,
	llm adapters.LLMClient,
	model string,
	toolCalls []toolCallEntry,
	answer string,
) (patterns []string, actions []string, errs []string) {
	detectors := []func([]toolCallEntry, string) (pattern string, action string, err error){
		detectSequentialValidateGenerate,
		detectRepeatedBashPattern,
		detectMissingSkill,
	}

	for _, detect := range detectors {
		pattern, action, err := detect(toolCalls, answer)
		if err != nil {
			errs = append(errs, fmt.Sprintf("detector %T: %v", detect, err))
			continue
		}
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
		if action != "" {
			actions = append(actions, action)
		}
	}
	return
}

// toolCallEntry is a flattened view of a tool call + its output.
type toolCallEntry struct {
	Name      string
	Args      string
	OutputLen int // characters of the output, for effort estimation
}

// extractToolCalls pulls tool calls from the last assistant message block
// in inputItems, and pairs them with their output.
// Only considers the most recent assistant turn to keep reflection fast.
func extractToolCalls(items []responses.ResponseInputItemUnionParam) []toolCallEntry {
	var calls []toolCallEntry
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		// Scan forward from this item to collect all function_call + output pairs.
		if item.OfFunctionCall != nil {
			calls = append([]toolCallEntry{{
				Name: item.OfFunctionCall.Name,
				Args: string(item.OfFunctionCall.Arguments),
			}}, calls...)
		}
		if item.OfFunctionCallOutput != nil {
			if len(calls) > 0 {
				// Output is a union type; use fmt.Sprintf to get a reasonable representation.
				calls[0].OutputLen = len(fmt.Sprintf("%v", item.OfFunctionCallOutput.Output))
			}
		}
	}
	return calls
}

// ─────────────────────────────────────────────────────────────────────────────
// Pattern 1: sequential validate + generate (the auto-chain gap)
// Detects: bash(make validate) → bash(make generate) in same turn.
// Fix:    patch the handler to auto-chain.
// ─────────────────────────────────────────────────────────────────────────────

var validateGenerateRE = regexp.MustCompile(`(?i)make\s+(validate|gen)`)

func detectSequentialValidateGenerate(calls []toolCallEntry, _ string) (pattern string, action string, err error) {
	if len(calls) < 2 {
		return "", "", nil
	}
	var validateIdx, generateIdx = -1, -1
	for i, c := range calls {
		if validateGenerateRE.MatchString(c.Args) {
			if strings.Contains(c.Args, "validate") || strings.Contains(c.Args, "make validate") {
				validateIdx = i
			}
			if strings.Contains(c.Args, "generate") || strings.Contains(c.Args, "make generate") {
				generateIdx = i
			}
		}
	}
	if validateIdx >= 0 && generateIdx >= 0 && validateIdx < generateIdx {
		pattern = "sequential validate+generate in same turn (auto-chain gap)"
		// Try to patch the handler. This is the most impactful fix.
		if fixed, fixErr := tryAutoChainFix(); fixErr == nil && fixed {
			action = "auto-chain added to handler (validate→generate now automatic)"
		} else if fixErr != nil {
			action = fmt.Sprintf("auto-chain fix attempted but: %v", fixErr)
		} else {
			action = "auto-chain already present or fix not applicable"
		}
	}
	return
}

// tryAutoChainFix attempts to add auto-chain SSE broadcast to handler files.
// It is conservative: only patches files where the sse package is already imported,
// otherwise records the pattern for manual follow-up.
func tryAutoChainFix() (bool, error) {
	handlersDir := "/root/vibex-workbench/agent/agents/runtime/tools"
	entries, err := os.ReadDir(handlersDir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(handlersDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		s := string(content)
		hasValidate := strings.Contains(s, "make validate")
		hasGenerate := strings.Contains(s, "make generate")
		hasSSEImport := strings.Contains(s, `"sse"`) || strings.Contains(s, `sse `)
		hasAutoChain := strings.Contains(s, "sse.Broadcast") ||
			strings.Contains(s, "Broadcaster") ||
			strings.Contains(s, "canvas_update")
		if hasValidate && hasGenerate && !hasAutoChain {
			if hasSSEImport {
				return patchHandlerAutoChain(path)
			}
			// sse not imported yet; skip to avoid breaking the build.
			// The pattern is still recorded by the caller.
		}
	}
	return false, nil
}

func patchHandlerAutoChain(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	s := string(content)

	// If already has SSE broadcast, nothing to do.
	if strings.Contains(s, "sse.Broadcast") || strings.Contains(s, "Broadcaster") {
		return false, nil
	}

	// Find the last "make generate" line and insert SSE broadcast after it.
	genLineIdx := strings.LastIndex(s, "make generate")
	if genLineIdx == -1 {
		return false, nil
	}
	// Find end of that line.
	lineEnd := strings.Index(s[genLineIdx:], "\n")
	if lineEnd == -1 {
		return false, nil
	}
	insertAt := genLineIdx + lineEnd

	sseSnippet := "\n\t// Self-reflection: auto-chain canvas_update (added " + time.Now().Format(time.RFC3339) + ")\n\tsse.Broadcast(\"canvas.spec_created\", map[string]string{\n\t\t\"source\": \"auto-chain\",\n\t})\n"

	newContent := s[:insertAt] + sseSnippet + s[insertAt:]
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return false, err
	}
	return true, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pattern 2: repeated bash command (3+ times same command)
// Detects: same bash command called N times in a turn.
// Fix:    suggest adding a skill or flag, log as actionable pattern.
// ─────────────────────────────────────────────────────────────────────────────

func detectRepeatedBashPattern(calls []toolCallEntry, _ string) (pattern string, action string, err error) {
	bashCount := 0
	for _, c := range calls {
		if c.Name == "bash" {
			bashCount++
		}
	}
	if bashCount >= 3 {
		pattern = fmt.Sprintf("high bash count in single turn (%d)", bashCount)
		action = "consider consolidating bash steps or adding a skill"
	}
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// Pattern 3: missing skill
// Detects: agent called a tool N times with the same name.
// Fix:    suggest creating a skill for it.
// ─────────────────────────────────────────────────────────────────────────────

func detectMissingSkill(calls []toolCallEntry, _ string) (pattern string, action string, err error) {
	// Count tool invocations.
	count := map[string]int{}
	for _, c := range calls {
		count[c.Name]++
	}
	for name, n := range count {
		if n >= 3 && name != "bash" {
			pattern = fmt.Sprintf("repeated tool '%s' %d times — could be a skill", name, n)
			action = fmt.Sprintf("suggest adding .skills/%s skill", name)
		}
	}
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// Self-reflection via LLM (optional, for complex pattern detection).
// Kept lightweight — only fires on turns with 5+ tool calls.
// ─────────────────────────────────────────────────────────────────────────────

var reflectPrompt = `You are a self-improvement agent. Analyze the tool calls from your last turn.

For each tool call, note:
- tool name and arguments
- whether it follows a repetitive pattern

If you notice:
1. Sequential pattern: create→validate→generate that was done manually
   → Return: {"fix": "auto_chain", "description": "..."}
2. Same bash command repeated 3+ times
   → Return: {"fix": "consolidate", "description": "..."}
3. Tool called repeatedly without a skill wrapper
   → Return: {"fix": "add_skill", "description": "..."}

Otherwise return: {"fix": null}

Respond with only valid JSON.`

func reflectWithLLM(ctx context.Context, llm adapters.LLMClient, model string, calls []toolCallEntry) (string, error) {
	if len(calls) < 5 {
		return "", nil // skip for simple turns
	}
	callsJSON, _ := json.MarshalIndent(calls, "", "  ")
	msgs := []responses.ResponseInputItemUnionParam{
		responses.ResponseInputItemParamOfMessage(reflectPrompt, responses.EasyInputMessageRoleDeveloper),
		responses.ResponseInputItemParamOfMessage(
			fmt.Sprintf("Tool calls from this turn:\n```\n%s\n```", string(callsJSON)),
			responses.EasyInputMessageRoleUser,
		),
	}
	reply, err := llm.SimpleChat(ctx, model, msgs)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(reply), nil
}

// RunSelfReflectionIfWorthy is called from the main loop after runToolLoop.
// It returns a summary string to optionally print.
func RunSelfReflectionIfWorthy(
	ctx context.Context,
	llm adapters.LLMClient,
	model string,
	inputItems []responses.ResponseInputItemUnionParam,
	answer string,
) string {
	result := reflectAfterTurn(ctx, llm, model, inputItems, answer)

	var lines []string
	for _, p := range result.PatternsFound {
		lines = append(lines, fmt.Sprintf("  [reflection] pattern: %s", p))
	}
	for _, a := range result.ActionsTaken {
		lines = append(lines, fmt.Sprintf("  [reflection] action: %s", a))
	}
	for _, e := range result.Errors {
		lines = append(lines, fmt.Sprintf("  [reflection] error: %s", e))
	}

	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("[self-reflection]\n%s", strings.Join(lines, "\n"))
}
