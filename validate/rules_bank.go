package validate

import (
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const BalanceFollowsTheBankRule RuleName = "balance-follows-bank"

func BalanceFollowsTheBank(
	balanceSlot tsv.Slot, yearColumn, cashColumn tsv.ColumnName,
	bankSlot tsv.Slot, bankYearColumn, bankAmountColumn tsv.ColumnName,
) Rule {
	return Rule{
		Name:  BalanceFollowsTheBankRule,
		Needs: []tsv.Slot{balanceSlot, bankSlot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			balances, banks := tables[balanceSlot], tables[bankSlot]

			atBalance, found := columnsOf(balances, balanceSlot, yearColumn, cashColumn)
			atBank, missing := columnsOf(banks, bankSlot, bankYearColumn, bankAmountColumn)
			found = append(found, missing...)
			if len(found) > 0 {
				return found
			}

			held := make(map[string]money.Yen, len(banks.Rows))
			for row, fields := range banks.Rows {
				written := fields[atBank[bankAmountColumn]]
				year := fields[atBank[bankYearColumn]]
				if isBlank(written) {
					if _, seen := held[year]; !seen {
						held[year] = 0
					}
					continue
				}
				amount, bad := yenAt(bankSlot, row, bankAmountColumn, written)
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				held[year] += amount
			}
			if len(found) > 0 {
				return found
			}

			for row, fields := range balances.Rows {
				year := fields[atBalance[yearColumn]]
				want, covered := held[year]
				if !covered {
					continue
				}

				cash, bad := yenAt(balanceSlot, row, cashColumn, fields[atBalance[cashColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				if cash != want {
					found = append(found, boundColumnFinding(
						balanceSlot, year, cashColumn, cash, "各行の年末残高の合計", want))
				}
			}
			return found
		},
	}
}
