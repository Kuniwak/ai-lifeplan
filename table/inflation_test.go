package table_test

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func onlyInflated(items ...input.PricedItem) map[input.PricedItem]money.PriceMove {
	answers := table.UniformInflationRatios(money.NewRate(0, 1))
	for _, item := range items {
		if !slices.Contains(input.PricedItems, item) {
			panic("計画に場所の無い費目: " + string(item))
		}
		answers[item] = money.RatioMove(money.NewRate(1, 1))
	}
	return answers
}

func expenseInputAtTwoPercent(t *testing.T) table.ExpenseInput {
	t.Helper()

	in, err := table.ExpenseInputFrom(tablesOfTheBaseProject(t), calendarOfTheBaseProject(t))
	if err != nil {
		t.Fatalf("table.ExpenseInputFrom: %v", err)
	}
	in.Loan, err = theLoanOfTheBaseProject.LoanTable(nil, planStart, planEnd, theFloatingOfTheBaseProject(planStart, planEnd))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}
	in.PriceLevelsByItem = levelsAt(t, twoPercentFrom2026(), table.UniformInflationRatios(money.NewRate(1, 1)))
	return in
}

func rateTable(from, to date.Year, at func(date.Year) money.Rate) relation.Table[money.Rate] {
	return relation.Over(relation.Span(from, to), at)
}

func TestPriceLevelsShouldStartAtOneInTheFirstYear(t *testing.T) {
	rates := rateTable(2018, 2020, func(date.Year) money.Rate { return money.NewRate(2, 100) })

	levels := table.PriceLevels(rates)

	first, ok := levels.At(2018)
	if !ok {
		t.Fatalf("最初の年の物価が無い")
	}
	if got, want := first.Apply(1_000_000), money.Yen(1_000_000); got != want {
		t.Errorf("基準年の 100 万円は %v のはずが %v", want, got)
	}
}

func twoPercentFrom2026() relation.Table[money.Rate] {
	return rateTable(planStart, planEnd, func(y date.Year) money.Rate {
		if y < 2026 {
			return money.NewRate(0, 100)
		}
		return money.NewRate(2, 100)
	})
}

func levelsAt(
	t *testing.T, rates relation.Table[money.Rate], ratios map[input.PricedItem]money.PriceMove,
) map[input.PricedItem]relation.Table[money.Factor] {
	t.Helper()

	levels, err := table.PriceLevelsByItem(rates, ratios)
	if err != nil {
		t.Fatalf("table.PriceLevelsByItem: %v", err)
	}
	return levels
}
