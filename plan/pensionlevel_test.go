package plan_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/plan"
)

func buildOrFail(t *testing.T, sources plan.Sources) *plan.Plan {
	t.Helper()
	built, err := plan.Build(sources)
	if err != nil {
		t.Fatalf("plan.Build: %v", err)
	}
	return built
}
