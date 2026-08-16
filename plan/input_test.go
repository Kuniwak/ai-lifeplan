package plan_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func theBaseInput(t *testing.T) *plan.Input {
	t.Helper()

	in, err := plan.Load(plan.Sources{Root: "..", ProjectPath: "../projects/classic.tsv"})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	return in
}

func aRate(percent string) *tsv.Table {
	return &tsv.Table{
		Header: []tsv.ColumnName{input.YearColumn, "実質運用利率"},
		Rows:   [][]string{{"2018", percent}},
	}
}

func TestWithShouldNotChangeTheInputItWasCalledOn(t *testing.T) {
	in := theBaseInput(t)

	before, err := in.Table(input.InvestmentReturnSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}

	in.With(input.InvestmentReturnSlot, aRate("9.99%"))

	after, err := in.Table(input.InvestmentReturnSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}
	if after != before {
		t.Error("With を呼んだあとで元の Input の表が変わっている")
	}
}

func TestStartsAfterShouldComeFromTheActuals(t *testing.T) {
	got, err := theBaseInput(t).StartsAfter()
	if err != nil {
		t.Fatalf("plan.Input.StartsAfter: %v", err)
	}
	if got < 2018 || got > 2100 {
		t.Errorf("実績の最終年が %d で、計画の範囲の外にある", got)
	}
}

func TestTableShouldRefuseASlotNothingFills(t *testing.T) {
	if _, err := theBaseInput(t).Table("no_such_slot"); err == nil {
		t.Error("埋まっていないスロットを求めて error にならない")
	}
}
