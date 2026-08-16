package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type TimelineRow struct {
	Receipts money.Yen

	Spending money.Yen

	SocialInsurance, Tax money.Yen

	Balance money.Yen
}

type TimelineInput struct {
	Income map[PersonName]relation.Table[IncomeRow]

	ChildAllowance relation.Table[ChildAllowanceRow]

	Expense         relation.Table[ExpenseRow]
	SocialInsurance relation.Table[SocialInsuranceTotalRow]
	Tax             relation.Table[TaxTotalRow]
}

func TimelineTable(in TimelineInput) (relation.Table[TimelineRow], error) {
	var empty relation.Table[TimelineRow]

	years := in.Expense.Years()
	rows := make([]relation.Row[TimelineRow], 0, len(years))

	for _, y := range years {
		var row TimelineRow

		var businessExpenses money.Yen
		for person, table := range in.Income {
			income, ok := table.At(y)
			if !ok {
				return empty, fmt.Errorf("table.TimelineTable: %q has no income for %d", person, y)
			}
			row.Receipts += income.Total
			businessExpenses += income.BusinessExpenses
		}

		allowance, ok := in.ChildAllowance.At(y)
		if !ok {
			return empty, fmt.Errorf("table.TimelineTable: no child allowance for %d", y)
		}
		row.Receipts += allowance.Total

		expense, ok := in.Expense.At(y)
		if !ok {
			return empty, fmt.Errorf("table.TimelineTable: no spending for %d", y)
		}
		row.Spending = expense.Total

		row.Spending += businessExpenses

		social, ok := in.SocialInsurance.At(y)
		if !ok {
			return empty, fmt.Errorf("table.TimelineTable: no social insurance for %d", y)
		}
		row.SocialInsurance = social.Total

		tax, ok := in.Tax.At(y)
		if !ok {
			return empty, fmt.Errorf("table.TimelineTable: no tax for %d", y)
		}
		row.Tax = tax.Total

		row.Balance = row.Receipts - row.Spending - row.SocialInsurance - row.Tax

		rows = append(rows, relation.Row[TimelineRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}

func (r TimelineRow) TakeHome() money.Yen {
	return r.Receipts - r.SocialInsurance - r.Tax
}
