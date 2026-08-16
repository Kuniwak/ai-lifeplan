package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const ChildcareLeaveBenefitTableName = "national/childcare-leave-benefit"

const (
	ChildcareLeaveRateColumn tsv.ColumnName = "支給率"
	ChildcareLeaveCapColumn  tsv.ColumnName = "支給上限額[円/月]"
)

const (
	ChildcareLeaveHigherRateMonths = 6
)

type ChildcareLeaveRate string

const (
	ChildcareLeaveHigherRate ChildcareLeaveRate = "67%"
	ChildcareLeaveLowerRate  ChildcareLeaveRate = "50%"
)

type ChildcareLeaveBenefitTable struct {
	caps map[ChildcareLeaveRate]YearYenTable
}

func ParseChildcareLeaveBenefitTable(table *tsv.Table) (ChildcareLeaveBenefitTable, error) {
	r, err := newReader(table, ChildcareLeaveBenefitTableName, ChildcareLeaveRateColumn, ChildcareLeaveCapColumn)
	if err != nil {
		return ChildcareLeaveBenefitTable{}, fmt.Errorf("law.ParseChildcareLeaveBenefitTable: %w", err)
	}

	byRate := make(map[ChildcareLeaveRate][]int, 2)
	for row := range r.Rows() {
		rate := ChildcareLeaveRate(r.Field(row, ChildcareLeaveRateColumn))
		byRate[rate] = append(byRate[rate], row)
	}

	caps := make(map[ChildcareLeaveRate]YearYenTable, len(byRate))
	for rate, rows := range byRate {
		only, err := rowsOf(table, rows)
		if err != nil {
			return ChildcareLeaveBenefitTable{}, fmt.Errorf("law.ParseChildcareLeaveBenefitTable: %s: %w", rate, err)
		}
		parsed, err := ParseYearYenTable(only, ChildcareLeaveBenefitTableName, ChildcareLeaveCapColumn)
		if err != nil {
			return ChildcareLeaveBenefitTable{}, fmt.Errorf("law.ParseChildcareLeaveBenefitTable: %s: %w", rate, err)
		}
		caps[rate] = parsed
	}

	for _, rate := range []ChildcareLeaveRate{ChildcareLeaveHigherRate, ChildcareLeaveLowerRate} {
		if _, ok := caps[rate]; !ok {
			return ChildcareLeaveBenefitTable{}, fmt.Errorf(
				"law.ParseChildcareLeaveBenefitTable: no rows for the rate %q, so a leave that reaches it could not be worked out", rate)
		}
	}

	return ChildcareLeaveBenefitTable{caps: caps}, nil
}

func (t ChildcareLeaveBenefitTable) Benefit(monthlyPay money.Yen, months int, year date.Year) (money.Yen, error) {
	if months < 0 {
		return 0, fmt.Errorf("law.ChildcareLeaveBenefitTable.Benefit: %d months of leave", months)
	}
	if monthlyPay < 0 {
		return 0, fmt.Errorf("law.ChildcareLeaveBenefitTable.Benefit: the pay is negative: %d", monthlyPay)
	}

	var total money.Yen
	for month := 1; month <= months; month++ {
		rate := ChildcareLeaveHigherRate
		if month > ChildcareLeaveHigherRateMonths {
			rate = ChildcareLeaveLowerRate
		}

		fraction, err := money.ParsePercent(string(rate))
		if err != nil {
			return 0, fmt.Errorf("law.ChildcareLeaveBenefitTable.Benefit: %w", err)
		}

		paid := monthlyPay.Mul(fraction, money.Truncate)
		total += min(paid, t.caps[rate].Amount(year))
	}
	return total, nil
}

func rowsOf(table *tsv.Table, rows []int) (*tsv.Table, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("law.rowsOf: no rows were named, so the table would be empty")
	}

	only := &tsv.Table{Header: table.Header}
	for _, row := range rows {
		only.Rows = append(only.Rows, table.Rows[row])
	}
	return only, nil
}
