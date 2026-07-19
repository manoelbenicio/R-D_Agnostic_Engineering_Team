package deploy

import (
	"testing"
)

func TestRolloutTriggersResolveToRunbooksAndNoRequiredAreOrphaned(t *testing.T) {
	plan := DefaultRolloutPlan()
	catalog := DefaultOperationsCatalog()

	runbookMap := make(map[RunbookID]bool)
	for _, rb := range catalog.Runbooks {
		if runbookMap[rb.ID] {
			t.Fatalf("duplicate runbook ID in catalog: %s", rb.ID)
		}
		runbookMap[rb.ID] = true
	}

	triggered := make(map[RunbookID]bool)
	for _, trigger := range plan.Triggers {
		if !runbookMap[trigger.Runbook] {
			t.Errorf("trigger %q references unknown runbook %q", trigger.ID, trigger.Runbook)
		}
		triggered[trigger.Runbook] = true
	}

	required := []RunbookID{RunbookRollback, RunbookIncident, RunbookEscalation}
	for _, req := range required {
		if !triggered[req] {
			t.Errorf("required runbook %q is orphaned (no trigger uses it)", req)
		}
	}
}
