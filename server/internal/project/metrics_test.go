package project

import "testing"

func TestMetrics(t *testing.T) {
	root := copyTestProject(t)
	p, err := LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	m := p.Metrics()
	if m.Total != 4 {
		t.Fatalf("expected total 4, got %d", m.Total)
	}
	if m.Done != 2 || m.InProgress != 2 {
		t.Fatalf("expected 2 done, 2 in_progress, got done=%d in_progress=%d", m.Done, m.InProgress)
	}
	if m.CompletionPercent != 50 {
		t.Fatalf("expected 50%%, got %d", m.CompletionPercent)
	}
	if m.AcceptanceCriteriaTotal != 3 || m.AcceptanceCriteriaComplete != 2 {
		t.Fatalf("expected AC 2/3, got %d/%d", m.AcceptanceCriteriaComplete, m.AcceptanceCriteriaTotal)
	}
}

func TestMetricsEmptyTaskboard(t *testing.T) {
	root := t.TempDir()
	p := &Project{Name: "Empty", Root: root, taskboard: Taskboard{Project: "Empty"}}
	m := p.Metrics()
	if m.Total != 0 || m.CompletionPercent != 0 {
		t.Fatalf("expected zero metrics, got %+v", m)
	}
}
