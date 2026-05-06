package designkit

import (
	"strings"
	"testing"
)

const minimalGoodYAML = `
spec:
  name: test
prototype:
  intent:
    target_user: traders
    business_goal: see risk
    primary_action: open dashboard
    success_criteria: risk visible in 5s
  ui_spec:
    page_purpose: dashboard
    sections:
      - name: Risk
        component_type: AlertCard
        priority: high
    states: [loading, empty, error, normal]
    responsive:
      desktop: sidebar
      mobile: stacked
    acceptance:
      - risk shows on load
`

func TestEvaluatePrototypeGateYAML_MinimalGood(t *testing.T) {
	ok, det := EvaluatePrototypeGateYAML([]byte(minimalGoodYAML))
	if !ok || det != nil {
		t.Fatalf("expected pass, got ok=%v det=%v", ok, det)
	}
}

func TestEvaluatePrototypeGateYAML_MissingIntent(t *testing.T) {
	y := strings.Replace(minimalGoodYAML, "target_user: traders", "target_user: \"\"", 1)
	ok, det := EvaluatePrototypeGateYAML([]byte(y))
	if ok || det == nil || len(det.Codes) == 0 {
		t.Fatalf("expected fail")
	}
}
