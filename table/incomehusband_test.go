package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestTotalIncomeShouldNeverExceedTotalReceipts(t *testing.T) {
	for _, row := range husbandIncomeOfTheBaseProject(t).Rows() {
		if row.Value.TotalIncome > row.Value.Total {
			t.Errorf("%d: 合計所得 %d is more than 総収入 %d",
				row.Year, row.Value.TotalIncome, row.Value.Total)
		}
		if row.Value.Total < 0 {
			t.Errorf("%d: 総収入 %d; receipts cannot be negative", row.Year, row.Value.Total)
		}
	}
}

func husbandIncomeOfTheBaseProject(t *testing.T) relation.Table[table.IncomeRow] {
	t.Helper()
	return incomeOfTheBaseProject(t, "夫", input.IncomeHusbandSlot)
}

func thePlansPensionOf(person table.PersonName) (basic, proportional money.Yen) {
	switch person {
	case "夫":
		return 800_000, 1_500_000
	case "妻":
		return 800_000, 200_000
	}
	return 0, 0
}

func incomeOfTheBaseProject(t *testing.T, person table.PersonName, paySlot tsv.Slot) relation.Table[table.IncomeRow] {
	t.Helper()

	tables := tablesOfTheBaseProject(t)
	calendar := calendarOfTheBaseProject(t)

	in, err := table.IncomeInputFor(tables, calendar, person, paySlot)
	if err != nil {
		t.Fatalf("table.IncomeInputFor: %v", err)
	}

	benefit := law.MustLoadChildcareLeaveBenefits(t, os.DirFS("../"+law.LawDirectory))
	in.ChildcareLeave = benefit

	in.Pension.Basic, in.Pension.Proportional = thePlansPensionOf(person)

	built, err := table.IncomeTable(in)
	if err != nil {
		t.Fatalf("table.IncomeTable: %v", err)
	}
	return built
}

func TestThePensionShouldScaleItsTwoPartsSeparately(t *testing.T) {
	p := table.Pension{
		StartYear:    2059,
		Basic:        1_000_000,
		Proportional: 2_000_000,
		Supplement:   0,
		Expected:     money.NewRate(100, 100),
	}

	for _, c := range []struct {
		name        string
		basic, prop money.Rate
		want        money.Yen
	}{
		{name: "水準が動かなければそのまま", basic: money.NewRate(100, 100), prop: money.NewRate(100, 100), want: 3_000_000},
		{name: "基礎だけ下がる", basic: money.NewRate(803, 1000), prop: money.NewRate(100, 100), want: 2_803_000},
		{name: "報酬比例だけ上がる", basic: money.NewRate(100, 100), prop: money.NewRate(1145, 1000), want: 3_290_000},
		{name: "過去30年投影ケース", basic: money.NewRate(803, 1000), prop: money.NewRate(1145, 1000), want: 3_093_000},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := p.AtLevel(table.PensionLevel{Basic: c.basic, Proportional: c.prop}).Received(2059)
			if got != c.want {
				t.Errorf("%d 円。%d 円のはず", got, c.want)
			}
		})
	}
}
