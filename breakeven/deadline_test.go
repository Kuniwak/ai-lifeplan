package breakeven_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/breakeven"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
)

func theDepletingProject(t *testing.T) *plan.Input {
	t.Helper()

	in, err := plan.Load(plan.Sources{
		Root: "..", ProjectPath: "../projects/case-zero-growth-as-now.tsv",
	})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	return in
}

func TestDeadlineShouldRefuseToLookAtYearsAlreadyOnRecord(t *testing.T) {
	in := theBaseProject(t)
	from := sweepsFrom(t, in)
	dial := dialOf(input.LivingCostSlot, "生活費[円/月]", from)

	_, err := breakeven.Deadline(in, dial, breakeven.YenSetting(200_000), from-1)

	if err == nil {
		t.Fatal("実績のある年を見ようとしたのに通った")
	}
}

func TestDeadlineShouldRefuseAPostponeUntilPastTheDialsOwnLastRow(t *testing.T) {
	in := theDepletingProject(t)
	from := sweepsFrom(t, in)
	dial, err := breakeven.PostponeDialOf(input.IncomeHusbandSlot, from)
	if err != nil {
		t.Fatalf("breakeven.PostponeDialOf: %v", err)
	}
	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}
	lastRow, err := dial.LastRowYear(written)
	if err != nil {
		t.Fatalf("breakeven.Dial.LastRowYear: %v", err)
	}

	_, err = breakeven.Deadline(in, dial, breakeven.YearsSetting(5), lastRow+1)

	if err == nil {
		t.Fatal("最後の行より先まで見ようとしたのに通った")
	}
	if !strings.Contains(err.Error(), fmt.Sprint(lastRow)) {
		t.Errorf("誤りが最後の行の年（%d）を名指ししていない: %v", lastRow, err)
	}
}
