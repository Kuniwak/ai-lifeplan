package plan_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/plan"
)

func theProject(t *testing.T, path string) *plan.Plan {
	t.Helper()

	built, err := plan.Build(plan.Sources{Root: "..", ProjectPath: path})
	if err != nil {
		t.Fatalf("plan.Build(%s): %v", path, err)
	}
	return built
}
