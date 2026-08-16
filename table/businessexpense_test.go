package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func TestBusinessExpensesShouldComeOffTheBusinessIncome(t *testing.T) {
	pay := table.Pay{BusinessReceipts: 1_000_000, BusinessExpenses: 220_000}

	if got, want := pay.BusinessIncome(), money.Yen(780_000); got != want {
		t.Errorf("事業所得が %d。事業収入 1,000,000 − 必要経費 220,000 = %d のはず", got, want)
	}
}

func TestBusinessExpensesShouldLeaveTheCashAsWellAsTheIncome(t *testing.T) {
	const year = date.Year(2031)

	timelineOf := func(t *testing.T, expenses money.Yen) table.TimelineRow {
		t.Helper()

		one := func(v table.IncomeRow) relation.Table[table.IncomeRow] {
			return relation.New([]relation.Row[table.IncomeRow]{{Year: year, Value: v}})
		}
		built, err := table.TimelineTable(table.TimelineInput{
			Income: map[table.PersonName]relation.Table[table.IncomeRow]{
				"妻": one(table.IncomeRow{
					BusinessReceipts: 1_000_000,
					BusinessExpenses: expenses,
					BusinessIncome:   1_000_000 - expenses,
					Total:            1_000_000,
					TotalIncome:      1_000_000 - expenses,
				}),
			},
			ChildAllowance: relation.New([]relation.Row[table.ChildAllowanceRow]{{Year: year}}),
			Expense: relation.New([]relation.Row[table.ExpenseRow]{
				{Year: year, Value: table.ExpenseRow{Total: 3_000_000}},
			}),
			SocialInsurance: relation.New([]relation.Row[table.SocialInsuranceTotalRow]{{Year: year}}),
			Tax:             relation.New([]relation.Row[table.TaxTotalRow]{{Year: year}}),
		})
		if err != nil {
			t.Fatalf("table.TimelineTable: %v", err)
		}
		row, ok := built.At(year)
		if !ok {
			t.Fatalf("%d が無い", year)
		}
		return row
	}

	without := timelineOf(t, 0)
	with := timelineOf(t, 220_000)

	if with.Receipts != without.Receipts {
		t.Errorf("総収入が %d から %d に動いた。経費は収入を減らさない", without.Receipts, with.Receipts)
	}

	if got, want := with.Spending-without.Spending, money.Yen(220_000); got != want {
		t.Errorf("総支出の差が %d。必要経費 %d のはず", got, want)
	}

	if got, want := without.Balance-with.Balance, money.Yen(220_000); got != want {
		t.Errorf("収支の差が %d。必要経費 %d のはず", got, want)
	}
}
