package table_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func timelineOfTheBaseProject(t *testing.T) relation.Table[table.TimelineRow] {
	t.Helper()

	built, err := table.TimelineTable(table.TimelineInput{
		Income: map[table.PersonName]relation.Table[table.IncomeRow]{
			"夫": incomeOfTheBaseProject(t, "夫", input.IncomeHusbandSlot),
			"妻": incomeOfTheBaseProject(t, "妻", input.IncomeWifeSlot),
		},
		ChildAllowance:  childAllowanceOfTheBaseProject(t),
		Expense:         expenseOfTheBaseProject(t),
		SocialInsurance: socialInsuranceTotalOfTheBaseProject(t),
		Tax:             taxTotalOfTheBaseProject(t),
	})
	if err != nil {
		t.Fatalf("table.TimelineTable: %v", err)
	}
	return built
}

func TestTheBalanceIdentityShouldHoldForAnyFigures(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		yen := func(name string) money.Yen {
			return money.Yen(rapid.Int64Range(0, 1_000_000_000).Draw(t, name))
		}
		row := table.TimelineRow{
			Receipts:        yen("receipts"),
			Spending:        yen("spending"),
			SocialInsurance: yen("social"),
			Tax:             yen("tax"),
		}
		row.Balance = row.Receipts - row.Spending - row.SocialInsurance - row.Tax

		if got, want := row.TakeHome(), row.Balance+row.Spending; got != want {
			t.Fatalf("手取り %d is not 収支 %d plus 総支出 %d", got, row.Balance, row.Spending)
		}
	})
}
