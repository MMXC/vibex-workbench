package runtime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	rtools "vibex/agent/agents/runtime/tools"
	"vibex/agent/vibex/domain/rulesengine"

	"github.com/openai/openai-go/v3/responses"
)

// Extended pseudo–tool-call detection (fenced JSON, minimax tags, narrative fake writes).
// Kept with the filter so CLI and web share one policy.
var (
	pseudoToolCallPattern            = regexp.MustCompile("(?is)\\[/TOOL_CALL\\]|```json\\s*\\{[\\s\\S]*?\"action\"\\s*:\\s*\".+?\"[\\s\\S]*?\\}\\s*```|<minimax:tool_call[\\s\\S]*?</minimax:tool_call>|<invoke\\s+name=\\\"[^\\\"]+\\\">")
	pseudoNamedToolJSONPattern       = regexp.MustCompile("(?is)```json\\s*\\{[\\s\\S]*?\\\"tool\\\"\\s*:")
	pseudoOpenAIStyleToolJSONPattern = regexp.MustCompile("(?is)```json\\s*\\{[\\s\\S]*?\\\"type\\\"\\s*:\\s*\\\"function\\\"[\\s\\S]*?\\\"name\\\"\\s*:")
)

type filterFailType string

const (
	failToolMissing      filterFailType = "tool_missing"
	failSemanticAmbig    filterFailType = "intent_ambiguous"
	failPolicyBlocked    filterFailType = "policy_blocked"
	failExecutionFailed  filterFailType = "execution_failed"
	failGoalNotReached   filterFailType = "goal_not_reached"
)

type toolSpecMeta struct {
	required map[string]struct{}
}

type filterEngine struct {
	tools map[string]toolSpecMeta
}

type rejectedCall struct {
	item   responses.ResponseInputItemUnionParam
	reason string
	kind   filterFailType
}

func newFilterEngine(tools []responses.ToolUnionParam) *filterEngine {
	metas := make(map[string]toolSpecMeta)
	for _, t := range tools {
		if t.OfFunction == nil {
			continue
		}
		reqSet := map[string]struct{}{}
		if arr, ok := t.OfFunction.Parameters["required"].([]interface{}); ok {
			for _, x := range arr {
				if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
					reqSet[strings.TrimSpace(s)] = struct{}{}
				}
			}
		}
		metas[t.OfFunction.Name] = toolSpecMeta{required: reqSet}
	}
	return &filterEngine{tools: metas}
}

func (f *filterEngine) preflight(
	toolCalls []responses.ResponseInputItemUnionParam,
	handlers map[string]rtools.Handler,
) (accepted []responses.ResponseInputItemUnionParam, rejected []rejectedCall) {
	accepted = make([]responses.ResponseInputItemUnionParam, 0, len(toolCalls))
	rejected = make([]rejectedCall, 0)
	for _, item := range toolCalls {
		if item.OfFunctionCall == nil {
			accepted = append(accepted, item)
			continue
		}
		call := item.OfFunctionCall
		name := strings.TrimSpace(call.Name)
		if _, ok := handlers[name]; !ok {
			rejected = append(rejected, rejectedCall{
				item:   item,
				reason: fmt.Sprintf("unknown tool: %s", name),
				kind:   failToolMissing,
			})
			continue
		}
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			rejected = append(rejected, rejectedCall{
				item:   item,
				reason: "arguments must be valid JSON object",
				kind:   failSemanticAmbig,
			})
			continue
		}
		if meta, ok := f.tools[name]; ok && len(meta.required) > 0 {
			miss := missingRequired(args, meta.required)
			if len(miss) > 0 {
				rejected = append(rejected, rejectedCall{
					item:   item,
					reason: "missing required arguments: " + strings.Join(miss, ", "),
					kind:   failSemanticAmbig,
				})
				continue
			}
		}
		accepted = append(accepted, item)
	}
	return accepted, rejected
}

func extendedPseudoToolRepair(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return false
	}
	if pseudoToolCallPattern.MatchString(t) || pseudoNamedToolJSONPattern.MatchString(t) ||
		pseudoOpenAIStyleToolJSONPattern.MatchString(t) {
		return true
	}
	if strings.Contains(t, "```yaml") && (strings.Contains(t, "prototype:") || strings.Contains(t, "specs/")) {
		if strings.Contains(t, "写入完成") || strings.Contains(t, "已写入") ||
			strings.Contains(t, "文件路径") || strings.Contains(strings.ToLower(t), "written") {
			return true
		}
	}
	return false
}

func (f *filterEngine) needsRetryWithoutToolCalls(text string) (bool, string, filterFailType) {
	t := strings.TrimSpace(text)
	if t == "" {
		return true, "empty assistant response; produce explicit plan + tool calls", failGoalNotReached
	}
	upper := strings.ToUpper(t)
	if strings.Contains(upper, "[TOOL_CALL]") || strings.Contains(upper, "{TOOL =>") {
		return true, "detected pseudo tool-call text; must use structured function calls", failPolicyBlocked
	}
	if extendedPseudoToolRepair(t) {
		return true, "detected pseudo tool-call or narrative fake write; use structured function_call only", failPolicyBlocked
	}
	return false, "", ""
}

func (f *filterEngine) executionNeedsRepair(out string) (bool, string, filterFailType) {
	lo := strings.ToLower(strings.TrimSpace(out))
	switch {
	case strings.HasPrefix(lo, "unsupported tool"):
		return true, "tool unsupported at runtime", failToolMissing
	case strings.HasPrefix(lo, "invalid args"):
		return true, "tool arguments invalid", failSemanticAmbig
	case strings.HasPrefix(lo, "blocked:"):
		return true, "tool blocked by policy", failPolicyBlocked
	case strings.HasPrefix(lo, "error:"):
		return true, "tool execution error", failExecutionFailed
	default:
		return false, "", ""
	}
}

func buildRepairHint(kind filterFailType, details string) string {
	return fmt.Sprintf(
		"Filter failed (%s): %s\nPlease repair intent/args/tool choice, then produce structured function calls only.",
		kind, details,
	)
}

func filterFailToRulesEngine(kind filterFailType) string {
	switch kind {
	case failToolMissing:
		return rulesengine.FailureToolMissing
	case failSemanticAmbig:
		return rulesengine.FailureIntentAmbiguous
	case failPolicyBlocked:
		return rulesengine.FailurePolicyBlocked
	case failExecutionFailed:
		return rulesengine.FailureExecutionFailed
	case failGoalNotReached:
		return rulesengine.FailureGoalNotReached
	default:
		return rulesengine.FailureIntentAmbiguous
	}
}

func missingRequired(args map[string]interface{}, req map[string]struct{}) []string {
	miss := make([]string, 0)
	for k := range req {
		v, ok := args[k]
		if !ok || isEmptyValue(v) {
			miss = append(miss, k)
		}
	}
	sort.Strings(miss)
	return miss
}

func isEmptyValue(v interface{}) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}
