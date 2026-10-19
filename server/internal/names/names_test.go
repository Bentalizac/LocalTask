package names

import (
	"strings"
	"testing"
)

func TestBundledListIsUsable(t *testing.T) {
	if len(All) < 1000 {
		t.Fatalf("expected the bundled list to hold over 1000 names, got %d", len(All))
	}
	seen := make(map[string]bool, len(All))
	for _, name := range All {
		key := strings.ToLower(name)
		if seen[key] {
			t.Fatalf("duplicate name %q (names compare without case)", name)
		}
		seen[key] = true
	}
}

func TestPickSkipsTakenNames(t *testing.T) {
	pool := []string{"Rand", "Mat", "Perrin"}
	taken := map[string]bool{"rand": true, "Perrin": true}
	for i := 0; i < 20; i++ {
		if got := Pick(pool, taken, nil); got != "Mat" {
			t.Fatalf("expected the only free name, Mat, got %q", got)
		}
	}
}

func TestPickNumbersANameWhenAllAreTaken(t *testing.T) {
	pool := []string{"Rand"}
	first := func(int) int { return 0 }
	if got := Pick(pool, map[string]bool{"Rand": true}, first); got != "Rand-2" {
		t.Fatalf("expected Rand-2, got %q", got)
	}
	if got := Pick(pool, map[string]bool{"Rand": true, "rand-2": true}, first); got != "Rand-3" {
		t.Fatalf("expected Rand-3, got %q", got)
	}
}
