package designkit

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// GateFailureDetail is returned when prototype gate blocks confirm=true writes.
type GateFailureDetail struct {
	Codes      []string    `json:"codes"`
	Checks     []GateCheck `json:"checks"`
	NextAction string      `json:"nextAction"`
}

// GateCheck mirrors frontend ui-workflow-gate.ts.
type GateCheck struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func nonEmptyStr(v interface{}) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) != ""
}

func nonEmptyList(v interface{}) bool {
	if v == nil {
		return false
	}
	arr, ok := v.([]interface{})
	if !ok {
		return false
	}
	for _, x := range arr {
		if nonEmptyStr(x) {
			return true
		}
	}
	return false
}

func asMap(v interface{}) map[string]interface{} {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return m
}

func statesSatisfied(raw interface{}) bool {
	if raw == nil {
		return false
	}
	names := map[string]struct{}{}
	required := []string{"loading", "empty", "error", "normal"}
	switch t := raw.(type) {
	case []interface{}:
		for _, x := range t {
			if s, ok := x.(string); ok {
				names[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
			}
		}
	case map[string]interface{}:
		for k := range t {
			names[strings.ToLower(strings.TrimSpace(k))] = struct{}{}
		}
	case string:
		s := strings.ToLower(t)
		for _, st := range required {
			if strings.Contains(s, st) {
				names[st] = struct{}{}
			}
		}
	default:
		return false
	}
	for _, st := range required {
		if _, ok := names[st]; !ok {
			return false
		}
	}
	return true
}

func responsiveSatisfied(raw interface{}) bool {
	m := asMap(raw)
	if m == nil {
		return nonEmptyStr(raw)
	}
	d := nonEmptyStr(m["desktop"])
	mo := nonEmptyStr(m["mobile"])
	su := nonEmptyStr(m["summary"])
	return (d && mo) || su
}

// EvaluatePrototypeGateYAML returns ok=true when all checks pass.
func EvaluatePrototypeGateYAML(yamlBytes []byte) (ok bool, detail *GateFailureDetail) {
	var checks []GateCheck
	var root map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &root); err != nil {
		checks = append(checks, GateCheck{
			ID: "yaml_parse", Label: "Spec YAML 可解析", Passed: false,
			Detail: err.Error(),
		})
		ok, d := finalizeGateReturn(checks)
		return ok, d
	}

	proto := asMap(root["prototype"])
	var intent, uiSpec map[string]interface{}
	if proto != nil {
		intent = asMap(proto["intent"])
		uiSpec = asMap(proto["ui_spec"])
	}

	var goal interface{}
	if intent != nil {
		goal = intent["business_goal"]
		if goal == nil {
			goal = intent["goal"]
		}
	}

	checks = append(checks, GateCheck{ID: "intent_target_user", Label: "Intent：目标用户（prototype.intent.target_user）", Passed: intent != nil && nonEmptyStr(intent["target_user"])})
	checks = append(checks, GateCheck{ID: "intent_goal", Label: "Intent：业务目标（prototype.intent.business_goal 或 goal）", Passed: nonEmptyStr(goal)})
	checks = append(checks, GateCheck{ID: "intent_primary_action", Label: "Intent：核心动作（prototype.intent.primary_action）", Passed: intent != nil && nonEmptyStr(intent["primary_action"])})
	scOk := false
	if intent != nil {
		scOk = nonEmptyStr(intent["success_criteria"]) || nonEmptyList(intent["success_criteria"])
	}
	checks = append(checks, GateCheck{ID: "intent_success", Label: "Intent：成功标准（prototype.intent.success_criteria）", Passed: scOk})

	sectionOK := false
	if uiSpec != nil {
		if secs, ok := uiSpec["sections"].([]interface{}); ok && len(secs) > 0 {
			all := true
			for _, s := range secs {
				sm := asMap(s)
				if sm == nil || !nonEmptyStr(sm["name"]) {
					all = false
					break
				}
				ct := sm["component_type"]
				if ct == nil {
					ct = sm["component-type"]
				}
				if !nonEmptyStr(ct) {
					all = false
					break
				}
			}
			sectionOK = all
		}
	}

	checks = append(checks, GateCheck{ID: "ui_page_purpose", Label: "UI Spec：页面目的（prototype.ui_spec.page_purpose）", Passed: uiSpec != nil && nonEmptyStr(uiSpec["page_purpose"])})
	checks = append(checks, GateCheck{ID: "ui_sections", Label: "UI Spec：模块列表（prototype.ui_spec.sections）", Passed: sectionOK})
	checks = append(checks, GateCheck{ID: "ui_states", Label: "UI Spec：状态矩阵（prototype.ui_spec.states）", Passed: uiSpec != nil && statesSatisfied(uiSpec["states"])})
	checks = append(checks, GateCheck{ID: "ui_responsive", Label: "UI Spec：响应式（prototype.ui_spec.responsive）", Passed: uiSpec != nil && responsiveSatisfied(uiSpec["responsive"])})
	acOk := false
	if uiSpec != nil {
		acOk = nonEmptyStr(uiSpec["acceptance"]) || nonEmptyList(uiSpec["acceptance"])
	}
	checks = append(checks, GateCheck{ID: "ui_acceptance", Label: "UI Spec：验收要点（prototype.ui_spec.acceptance）", Passed: acOk})

	return finalizeGateReturn(checks)
}

func finalizeGateReturn(checks []GateCheck) (bool, *GateFailureDetail) {
	var failed []GateCheck
	for i := range checks {
		if !checks[i].Passed {
			failed = append(failed, checks[i])
		}
	}
	if len(failed) == 0 {
		return true, nil
	}
	codes := map[string]struct{}{}
	for _, f := range failed {
		if strings.HasPrefix(f.ID, "intent_") {
			codes["intent_incomplete"] = struct{}{}
		} else if f.ID == "ui_acceptance" {
			codes["acceptance_missing"] = struct{}{}
		} else if strings.HasPrefix(f.ID, "ui_") {
			codes["ui_spec_incomplete"] = struct{}{}
		}
	}
	outCodes := make([]string, 0, len(codes))
	for c := range codes {
		outCodes = append(outCodes, c)
	}
	next := buildNextActionGo(failed)
	return false, &GateFailureDetail{
		Codes:      outCodes,
		Checks:     checks,
		NextAction: next,
	}
}

func buildNextActionGo(failed []GateCheck) string {
	var b strings.Builder
	b.WriteString("请在当前 spec 的 prototype 下补齐 Intent / UI Spec（详见 gate checks）。未通过项：\n")
	for _, f := range failed {
		line := fmt.Sprintf("- %s", f.Label)
		if f.Detail != "" {
			line += "：" + f.Detail
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
