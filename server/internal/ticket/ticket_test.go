package ticket

import "testing"

func TestTypeValid(t *testing.T) {
	if !TypeTask.Valid() {
		t.Error("expected task to be valid")
	}
	if Type("nonsense").Valid() {
		t.Error("expected nonsense to be invalid")
	}
}

func TestStatusValid(t *testing.T) {
	if !StatusInProgress.Valid() {
		t.Error("expected in_progress to be valid")
	}
	if Status("nonsense").Valid() {
		t.Error("expected nonsense to be invalid")
	}
}

func TestPriorityValid(t *testing.T) {
	if !PriorityCritical.Valid() {
		t.Error("expected critical to be valid")
	}
	if Priority("nonsense").Valid() {
		t.Error("expected nonsense to be invalid")
	}
}

func TestAcceptanceCriteriaProgress(t *testing.T) {
	tk := Ticket{
		AcceptanceCriteria: []AcceptanceCriterion{
			{ID: "a1", Text: "one", Complete: true},
			{ID: "a2", Text: "two", Complete: false},
			{ID: "a3", Text: "three", Complete: true},
		},
	}
	complete, total := tk.AcceptanceCriteriaProgress()
	if complete != 2 || total != 3 {
		t.Fatalf("expected 2/3, got %d/%d", complete, total)
	}
}

func TestHasTag(t *testing.T) {
	tk := Ticket{Tags: []string{"backend", "video"}}
	if !tk.HasTag("backend") {
		t.Error("expected HasTag(backend) to be true")
	}
	if tk.HasTag("frontend") {
		t.Error("expected HasTag(frontend) to be false")
	}
}
