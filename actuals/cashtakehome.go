package actuals

import (
	"os"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

const MonthsInAYear = 12

type CashTakeHomeRow struct {
	TakeHome money.Yen

	Months int
}

func CashTakeHome(slips []Payslip, returns map[date.Year]TaxReturn) relation.Table[CashTakeHomeRow] {
	byYear := make(map[date.Year]CashTakeHomeRow, 16)
	salaryMonths := make(map[date.Year]map[int]bool, 16)
	for _, slip := range slips {
		year := date.Year(slip.Year)
		row := byYear[year]
		row.TakeHome += slip.Gross - slip.Health - slip.Pension -
			slip.Employment - slip.IncomeTax - slip.ResidentTax
		byYear[year] = row

		if slip.Kind == PayslipSalary {
			if salaryMonths[year] == nil {
				salaryMonths[year] = make(map[int]bool, MonthsInAYear)
			}
			salaryMonths[year][slip.Month] = true
		}
	}

	for year, row := range byYear {
		row.Months = len(salaryMonths[year])

		if previous, ok := returns[year-1]; ok && previous.Has("還付される税金") {
			row.TakeHome += previous["還付される税金"]
		}
		if this, ok := returns[year]; ok && this.Has("業務雑所得の収入") {
			row.TakeHome += this["業務雑所得の収入"] - this["業務雑所得の源泉徴収税額"]
		}
		byYear[year] = row
	}

	rows := make([]relation.Row[CashTakeHomeRow], 0, len(byYear))
	for year, row := range byYear {
		rows = append(rows, relation.Row[CashTakeHomeRow]{Year: year, Value: row})
	}
	return relation.New(rows)
}

func CashTakeHomeUnder(root string) (relation.Table[CashTakeHomeRow], error) {
	slips, err := PayslipsUnder(os.DirFS(root))
	if err != nil {
		return relation.Table[CashTakeHomeRow]{}, err
	}
	returns, err := ReadTaxReturnsUnder(root)
	if err != nil {
		return relation.Table[CashTakeHomeRow]{}, err
	}
	return CashTakeHome(slips, returns), nil
}
