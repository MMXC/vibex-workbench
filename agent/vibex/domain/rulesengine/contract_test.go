package rulesengine

import (
	"encoding/json"
	"testing"
)

func TestStrongValidationPlanJSONRoundTrip(t *testing.T) {
	p := StrongValidationPlan{
		PlanID:      "p1",
		SlotBinding: "prototype",
		SpecPath:    ".vibex/specs/foo.yaml",
		Items: []StrongValidationItem{
			{ID: "a", Command: "make validate", TimeoutSec: 120, ExpectSignal: "green"},
		},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var out StrongValidationPlan
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.PlanID != p.PlanID || len(out.Items) != 1 || out.Items[0].Command != "make validate" {
		t.Fatalf("unexpected: %#v", out)
	}
}

func TestAgentNodeRefAndTraceJSONRoundTrip(t *testing.T) {
	ref := AgentNodeRef{
		NodeID:  "n-plan-3",
		Kind:    NodeKindPlanGraphStep,
		GraphID: "pg-2026",
		Label:   "route tools",
	}
	env := RepairEnvelope{
		FailedNodeID:  ref.NodeID,
		FailedNodeRef: &ref,
		ErrorMessage:  "boom",
		FailureType:   FailureExecutionFailed,
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var out RepairEnvelope
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.FailedNodeRef == nil || out.FailedNodeRef.Kind != NodeKindPlanGraphStep {
		t.Fatalf("unexpected: %#v", out)
	}

	ev := OrchestrationTraceEvent{
		Phase:          PhaseRouteExecute,
		Node:           ref,
		RunID:          "run-1",
		ChildRunID:     "run-1a",
		OutcomeSummary: "ok",
	}
	b2, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var evOut OrchestrationTraceEvent
	if err := json.Unmarshal(b2, &evOut); err != nil {
		t.Fatal(err)
	}
	if evOut.Node.NodeID != ref.NodeID || evOut.Phase != PhaseRouteExecute {
		t.Fatalf("unexpected: %#v", evOut)
	}
}

func TestCDPValidationPlanAndOutcomeJSONRoundTrip(t *testing.T) {
	plan := CDPValidationPlan{
		PlanID: "cdp-1",
		TargetEnv: CDPTargetEnvRef{
			Deployment: TestEnvUserManaged,
			Host:       "127.0.0.1",
			Port:       9222,
			TimeoutSec: 30,
		},
		EntryURL: "http://localhost:5173/workbench",
		Steps: []CDPValidationStep{
			{
				ID: "s1",
				Assertions: []CDPAssertion{
					{ID: "a1", Type: "text_contains", Value: "Spec"},
				},
			},
		},
	}
	b, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var p2 CDPValidationPlan
	if err := json.Unmarshal(b, &p2); err != nil {
		t.Fatal(err)
	}
	if p2.PlanID != plan.PlanID || p2.TargetEnv.Port != 9222 || len(p2.Steps) != 1 {
		t.Fatalf("plan: %#v", p2)
	}
	out := CDPValidationOutcome{OK: false, Logs: []string{"navigate ok"}, Error: "assert failed"}
	b2, _ := json.Marshal(out)
	var o2 CDPValidationOutcome
	if err := json.Unmarshal(b2, &o2); err != nil {
		t.Fatal(err)
	}
	if o2.OK || o2.Error == "" {
		t.Fatalf("outcome: %#v", o2)
	}
}
