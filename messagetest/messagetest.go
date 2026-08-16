package messagetest

import (
	"strings"
	"testing"
)

func AssertAll(t *testing.T, got []string, want ...[]string) {
	t.Helper()

	missing, unmatched := Match(got, want)

	for _, parts := range missing {
		t.Errorf("%v すべてに触れる finding が無い（全部で %d 件）: %v", parts, len(got), got)
	}
	if len(unmatched) > 0 {
		t.Errorf("誰も待っていない finding が %d 件出た: %v\n出たのは全部で %d 件: %v", len(unmatched), unmatched, len(got), got)
	}
}

func Match(got []string, want [][]string) (missing [][]string, unmatched []string) {
	heldBy := make([]int, len(got))
	for i := range heldBy {
		heldBy[i] = -1
	}

	var take func(w int, asked []bool) bool
	take = func(w int, asked []bool) bool {
		for i, message := range got {
			if asked[i] || !containsAll(message, want[w]) {
				continue
			}
			asked[i] = true
			if heldBy[i] < 0 || take(heldBy[i], asked) {
				heldBy[i] = w
				return true
			}
		}
		return false
	}

	for w := range want {
		if !take(w, make([]bool, len(got))) {
			missing = append(missing, want[w])
		}
	}
	for i, message := range got {
		if heldBy[i] < 0 {
			unmatched = append(unmatched, message)
		}
	}
	return missing, unmatched
}

func AssertOne(t *testing.T, got []string, parts ...string) {
	t.Helper()

	if len(parts) == 0 {
		AssertAll(t, got)
		return
	}
	AssertAll(t, got, parts)
}

func containsAll(message string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(message, part) {
			return false
		}
	}
	return true
}
