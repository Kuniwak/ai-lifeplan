package plan

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func theCellThatRunsOut(t *testing.T) *Input {
	t.Helper()

	in, err := Load(Sources{Root: "..", ProjectPath: "../projects/base.tsv",
		SlotOverrides: map[tsv.Slot]string{
			"inflation":         "data/environment/scenario/inflation-zero-growth.tsv",
			"real_wage_growth":  "data/environment/scenario/wage-zero-growth.tsv",
			"investment_return": "data/environment/scenario/return-zero-growth.tsv",
			"pension_level":     "data/environment/scenario/pension-zero-growth-depleted.tsv",
			"income_husband":    "data/controllable/income-husband.tsv",
			"living_cost":       "data/controllable/scenario/living-cost-32.tsv",
		}})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return in
}
