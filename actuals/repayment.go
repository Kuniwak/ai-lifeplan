package actuals

import (
	"fmt"
	"path/filepath"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const RepaymentPath tsv.Slot = "actuals/loan/repayment.tsv"

const (
	RepaymentContractColumn tsv.ColumnName = "契約"
	RepaymentYearColumn     tsv.ColumnName = "西暦"
	RepaymentPaidColumn     tsv.ColumnName = "返済合計[円]"
	RepaymentBalanceColumn  tsv.ColumnName = "年末残高[円]"
)

type RepaymentYear struct {
	Paid, Balance money.Yen

	Whole bool
}

func ReadRepaymentsUnder(root string) (map[date.Year]RepaymentYear, error) {
	read, err := tsv.ReadFile(filepath.Join(root, string(RepaymentPath)))
	if err != nil {
		return nil, err
	}
	return RepaymentsByYear(read)
}

func RepaymentsByYear(history *tsv.Table) (map[date.Year]RepaymentYear, error) {
	r, err := tsv.NewReader(history, RepaymentPath,
		RepaymentContractColumn, RepaymentYearColumn, RepaymentPaidColumn, RepaymentBalanceColumn)
	if err != nil {
		return nil, fmt.Errorf("actuals.RepaymentsByYear: %w", err)
	}

	contracts := map[string]bool{}
	rowsIn := map[date.Year]int{}
	byYear := map[date.Year]RepaymentYear{}
	for row := range r.Rows() {
		year, err := date.ParseYear(r.Field(row, RepaymentYearColumn))
		if err != nil {
			return nil, fmt.Errorf("actuals.RepaymentsByYear: %w",
				r.Errorf(row, RepaymentYearColumn, "%v", err))
		}
		paid, err := money.ParseYen(r.Field(row, RepaymentPaidColumn))
		if err != nil {
			return nil, fmt.Errorf("actuals.RepaymentsByYear: %w",
				r.Errorf(row, RepaymentPaidColumn, "%v", err))
		}
		balance, err := money.ParseYen(r.Field(row, RepaymentBalanceColumn))
		if err != nil {
			return nil, fmt.Errorf("actuals.RepaymentsByYear: %w",
				r.Errorf(row, RepaymentBalanceColumn, "%v", err))
		}

		contracts[r.Field(row, RepaymentContractColumn)] = true
		rowsIn[year]++
		sum := byYear[year]
		sum.Paid += paid
		sum.Balance += balance
		byYear[year] = sum
	}

	latest := date.Year(0)
	for year := range byYear {
		latest = max(latest, year)
	}
	for year, rows := range rowsIn {
		sum := byYear[year]
		sum.Whole = rows == len(contracts) && year != latest
		byYear[year] = sum
	}
	return byYear, nil
}
