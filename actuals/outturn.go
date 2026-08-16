package actuals

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Outturn struct {
	Movement money.Yen

	TakeHome money.Yen

	TakeHomeBasis TakeHomeBasis

	Outgoing money.Yen

	Cashflow money.Yen

	Unexplained money.Yen

	Partial bool
}

type TakeHomeBasis string

const (
	CashBasis TakeHomeBasis = "現金"

	AccrualBasis TakeHomeBasis = "発生"
)

type TakeHome struct {
	Value money.Yen
	Basis TakeHomeBasis
}

func Outturns(
	balances BalanceTable,
	takeHome relation.Table[TakeHome],
	cashflow relation.Table[money.Yen],
) (relation.Table[Outturn], []date.Year, error) {
	var empty relation.Table[Outturn]

	years := balances.Years()
	rows := make([]relation.Row[Outturn], 0, len(years))

	compared := make(map[date.Year]bool, len(years))
	for i, year := range years {
		if i == 0 {
			continue
		}
		compared[year] = true
	}
	var uncompared []date.Year
	for _, row := range cashflow.Rows() {
		if !compared[row.Year] {
			uncompared = append(uncompared, row.Year)
		}
	}

	for i, year := range years {
		if i == 0 {
			continue
		}

		if before := years[i-1]; year != before+1 {
			return empty, nil, fmt.Errorf(
				"actuals.Outturns: 残高の記録が %d から %d に飛んでいる。増減を 1 年ぶんとして扱えない",
				before, year)
		}

		closing, _ := balances.At(year)
		opening, _ := balances.At(years[i-1])

		earned, ok := takeHome.At(year)
		if !ok {
			return empty, nil, fmt.Errorf("actuals.Outturns: %d の手取りが分からない", year)
		}
		spent, ok := cashflow.At(year)
		if !ok {
			return empty, nil, fmt.Errorf("actuals.Outturns: %d の収支明細が無い", year)
		}

		row := Outturn{
			Movement:      closing.Total() - opening.Total(),
			TakeHome:      earned.Value,
			TakeHomeBasis: earned.Basis,
			Cashflow:      spent,
			Partial:       closing.Partial || opening.Partial,
		}
		row.Outgoing = row.TakeHome - row.Movement
		row.Unexplained = row.Movement - row.Cashflow

		rows = append(rows, relation.Row[Outturn]{Year: year, Value: row})
	}

	return relation.New(rows), uncompared, nil
}

func YearlyCashflow(table *tsv.Table) (relation.Table[money.Yen], error) {
	var empty relation.Table[money.Yen]

	r, err := tsv.NewReader(table, CashflowPath, CashflowMonthColumn, CashflowAmountColumn)
	if err != nil {
		return empty, fmt.Errorf("actuals.YearlyCashflow: %w", err)
	}

	byYear := make(map[date.Year]money.Yen, 8)
	for row := range r.Rows() {
		month := r.Field(row, CashflowMonthColumn)
		if len(month) < 4 {
			return empty, r.Errorf(row, CashflowMonthColumn, "%q is not a year and month", month)
		}
		year, err := date.ParseYear(month[:4])
		if err != nil {
			return empty, r.Errorf(row, CashflowMonthColumn, "%v", err)
		}
		amount, err := money.ParseYen(r.Field(row, CashflowAmountColumn))
		if err != nil {
			return empty, r.Errorf(row, CashflowAmountColumn, "%v", err)
		}
		byYear[year] += amount
	}

	rows := make([]relation.Row[money.Yen], 0, len(byYear))
	for year, total := range byYear {
		rows = append(rows, relation.Row[money.Yen]{Year: year, Value: total})
	}
	return relation.New(rows), nil
}
