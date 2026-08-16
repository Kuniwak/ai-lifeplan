package plan_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func theBaseProject(t *testing.T) *plan.Plan {
	t.Helper()

	built, err := plan.Build(plan.Sources{Root: "..", ProjectPath: "../projects/classic.tsv"})
	if err != nil {
		t.Fatalf("plan.Build: %v", err)
	}
	return built
}

func TestLoadShouldRefuseAManifestNamingATableThatIsNotThere(t *testing.T) {
	_, err := plan.Load(plan.Sources{
		Root:        "..",
		ProjectPath: "../projects/base.tsv",
		SlotOverrides: map[tsv.Slot]string{
			input.IncomeWifeSlot: "data/controllable/scenario/no-such-table.tsv",
		},
	})
	if err == nil {
		t.Fatal("無い表を名指したマニフェストが通った。**この先で nil として読まれ、遠いところで落ちる**")
	}
	for _, want := range []string{string(input.IncomeWifeSlot), "no-such-table.tsv"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラーが %q に触れていない: %v", want, err)
		}
	}
}
