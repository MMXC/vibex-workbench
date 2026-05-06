package rulesengine

import (
	"encoding/json"
	"testing"
)

func TestStrongValidationPlanJSONRoundTrip(t *testing.T) {
	p := StrongValidationPlan{
		PlanID:      "p1",
		SlotBinding: "prototype",
		SpecPath:    "specs/foo.yaml",
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
		Phase:        PhaseRouteExecute,
		Node:         ref,
		RunID:        "run-1",
		ChildRunID:   "run-1a",
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
