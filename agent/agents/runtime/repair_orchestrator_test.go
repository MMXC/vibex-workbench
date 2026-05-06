package runtime

import (
	"testing"

	"vibex/agent/vibex/domain/rulesengine"
)

func TestDecideRepair_policyBlocked(t *testing.T) {
	env := rulesengine.RepairEnvelope{
		FailureType:  rulesengine.FailurePolicyBlocked,
		ErrorMessage: "pseudo tool-call",
	}
	d := DecideRepair(env, 0)
	if d.TargetPhase != rulesengine.PhaseClarify || !d.Allowed {
		t.Fatalf("unexpected: %+v", d)
	}
}

func TestDecideRepair_exhausted(t *testing.T) {
	env := rulesengine.RepairEnvelope{FailureType: rulesengine.FailureGoalNotReached}
	d := DecideRepair(env, maxRepairAttempts)
	if d.Allowed || d.ReasonCode != "repair_exhausted" {
		t.Fatalf("unexpected: %+v", d)
	}
}
