package table_test

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
)

func TestEveryYenTheHouseholdWroteShouldBeInTheNet(t *testing.T) {
	for _, item := range []input.PricedItem{
		input.ContributionItem, input.CashFloorItem, input.MutualAidItem,
	} {
		if !slices.Contains(input.PricedItems, item) {
			t.Errorf("%q が適用可否の網の外にいる。世帯が円で書いた額はすべて答えを求められる", item)
		}
	}
}

func TestTheItemNamesShouldNotBeReadAsTheRenderedColumnNames(t *testing.T) {
	rendered := make([]string, 0, len(plan.ExpenseColumns))
	for _, c := range plan.ExpenseColumns {
		rendered = append(rendered, string(c))
	}

	wantAbsent := map[input.PricedItem]string{
		input.EducationItem:      "教育費合計 として合計で書き出される",
		input.LifeInsuranceItem:  "保険料合計 に畳まれ、部品としては現れない",
		input.FireInsuranceItem:  "保険料合計 に畳まれ、部品としては現れない",
		input.QuakeInsuranceItem: "保険料合計 に畳まれ、部品としては現れない",
		input.ContributionItem:   "支出ではない。資産の表に現れる",
		input.CashFloorItem:      "支出ではない。資産の表に現れる",
		input.MutualAidItem:      "支出ではない。資産の表に現れる",
	}

	for _, item := range input.PricedItems {
		present := slices.Contains(rendered, string(item))
		why, expected := wantAbsent[item]

		switch {
		case expected && present:
			t.Errorf("%q は %s はずなのに、支出の列に現れている。"+
				"表が変わったのならこの一覧を直すこと", item, why)
		case !expected && !present:
			t.Errorf("%q が支出の列に無い。列名が変わったか、項目を足して"+
				"この一覧に書き忘れたかのどちらかである", item)
		}
	}
}
