package validate

import (
	"fmt"
	"strconv"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	LoanSettlementInTermRule RuleName = "loan-settlement-in-term"
	LoanFixedPeriodRule      RuleName = "loan-fixed-period"
)

func LoanFixedPeriod(slot tsv.Slot, yearsColumn, fixedColumn tsv.ColumnName) Rule {
	return Rule{
		Name:  LoanFixedPeriodRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, yearsColumn, fixedColumn)
			if len(found) > 0 {
				return found
			}

			for row, fields := range table.Rows {
				years, err := wholeNumber(fields[at[yearsColumn]])
				if err != nil {
					found = append(found, unreadableField(slot, row, yearsColumn, err))
					continue
				}
				fixed, err := wholeNumber(fields[at[fixedColumn]])
				if err != nil {
					found = append(found, unreadableField(slot, row, fixedColumn, err))
					continue
				}
				if fixed <= 0 || fixed > years {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"row %d: %s が %d 年で、%s の %d 年に収まらない。全期間固定なら %d と書く",
							row+1, fixedColumn, fixed, yearsColumn, years, years),
					})
				}
			}
			return found
		},
	}
}

func wholeNumber(field string) (int64, error) {
	n, err := strconv.ParseInt(field, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("数でない: %q", field)
	}
	return n, nil
}

func LoanSettlementInTerm(
	loanSlot tsv.Slot, firstYearColumn, firstMonthColumn, yearsColumn tsv.ColumnName,
	settlementSlot tsv.Slot, settledColumn tsv.ColumnName, never string,
) Rule {
	return Rule{
		Name:  LoanSettlementInTermRule,
		Needs: []tsv.Slot{loanSlot, settlementSlot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			settlement := tables[settlementSlot]

			at, found := columnsOf(settlement, settlementSlot, settledColumn)
			if len(found) > 0 {
				return found
			}
			if len(settlement.Rows) != 1 {
				return []Finding{{
					Slot:    settlementSlot,
					Message: fmt.Sprintf("一行でなければならない。%d 行ある", len(settlement.Rows)),
				}}
			}

			written := settlement.Rows[0][at[settledColumn]]
			if written == never {
				return nil
			}
			settled, err := date.ParseYear(written)
			if err != nil {
				return []Finding{{Slot: settlementSlot, Message: err.Error()}}
			}

			loans := tables[loanSlot]
			terms, found := columnsOf(loans, loanSlot, firstYearColumn, firstMonthColumn, yearsColumn)
			if len(found) > 0 {
				return found
			}

			for row, fields := range loans.Rows {
				first, err := date.ParseYear(fields[terms[firstYearColumn]])
				if err != nil {
					found = append(found, unreadableField(loanSlot, row, firstYearColumn, err))
					continue
				}
				years, err := wholeNumber(fields[terms[yearsColumn]])
				if err != nil {
					found = append(found, unreadableField(loanSlot, row, yearsColumn, err))
					continue
				}

				month, err := wholeNumber(fields[terms[firstMonthColumn]])
				if err != nil {
					found = append(found, unreadableField(loanSlot, row, firstMonthColumn, err))
					continue
				}

				last := first + date.Year((month-1+years*12-1)/12)
				if settled < first || settled > last {
					found = append(found, Finding{
						Slot: settlementSlot,
						Message: fmt.Sprintf(
							"一括返済の年 %d が返済期間の外にある。row %d の契約は %d 年から %d 年まで返す",
							settled, row+1, first, last),
					})
				}
			}
			return found
		},
	}
}
