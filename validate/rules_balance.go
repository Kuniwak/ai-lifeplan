package validate

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	BalanceFollowsStatementsRule RuleName = "balance-follows-statements"
)

type BalancePockets struct {
	NISA, Taxable string
}

func BalanceFollowsTheStatements(
	balance SplitBalanceSide, statements StatementSide, outside OutsideSide, pockets BalancePockets,
) Rule {
	balanceSlot, holdingsSlot, outsideSlot := balance.Slot, statements.Slot, outside.Slot
	yearColumn, nisaColumn := balance.Year, balance.NISA
	taxableColumn, basisColumn := balance.Taxable, balance.Basis
	asOfColumn, pocketColumn := statements.AsOf, statements.Pocket
	valueColumn, gainColumn := statements.Value, statements.Gain
	outsideYearColumn, outsideAmountColumn, outsidePocketColumn := outside.Year, outside.Amount, outside.Pocket
	nisaPocket, taxablePocket := pockets.NISA, pockets.Taxable
	return Rule{
		Name:  BalanceFollowsStatementsRule,
		Needs: []tsv.Slot{balanceSlot, holdingsSlot, outsideSlot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			balances := tables[balanceSlot]
			holdings := tables[holdingsSlot]

			atBalance, found := columnsOf(balances, balanceSlot,
				yearColumn, nisaColumn, taxableColumn, basisColumn)
			atHolding, missing := columnsOf(holdings, holdingsSlot,
				asOfColumn, pocketColumn, valueColumn, gainColumn)
			found = append(found, missing...)
			atOutside, missing := columnsOf(tables[outsideSlot], outsideSlot,
				outsideYearColumn, outsideAmountColumn, outsidePocketColumn)
			found = append(found, missing...)
			if len(found) > 0 {
				return found
			}

			outside := make(map[string]map[string]money.Yen, len(tables[outsideSlot].Rows))
			for row, fields := range tables[outsideSlot].Rows {
				amount, bad := yenAt(outsideSlot, row, outsideAmountColumn, fields[atOutside[outsideAmountColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				pocket := fields[atOutside[outsidePocketColumn]]
				if pocket != nisaPocket && pocket != taxablePocket {
					found = append(found, Finding{
						Slot: outsideSlot,
						Message: fmt.Sprintf("row %d: %s が %q。%q か %q でなければ、どちらの枠に入るか決まらない",
							row+1, outsidePocketColumn, pocket, nisaPocket, taxablePocket),
					})
					continue
				}
				year := fields[atOutside[outsideYearColumn]]
				if _, seen := outside[year]; !seen {
					outside[year] = make(map[string]money.Yen, 2)
				}
				outside[year][pocket] += amount
			}
			if len(found) > 0 {
				return found
			}

			type pot struct{ value, basis money.Yen }
			said := make(map[string]map[string]pot)
			for row, fields := range holdings.Rows {
				asOf := fields[atHolding[asOfColumn]]
				year, isYearEnd := yearEnding(asOf)
				if !isYearEnd {
					continue
				}
				value, bad := yenAt(holdingsSlot, row, valueColumn, fields[atHolding[valueColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				gain, bad := yenAt(holdingsSlot, row, gainColumn, fields[atHolding[gainColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				if _, seen := said[year]; !seen {
					said[year] = make(map[string]pot)
				}
				at := said[year][fields[atHolding[pocketColumn]]]
				at.value += value
				at.basis += value - gain
				said[year][fields[atHolding[pocketColumn]]] = at
			}
			if len(found) > 0 {
				return found
			}

			for row, fields := range balances.Rows {
				year := fields[atBalance[yearColumn]]
				statement, covered := said[year]
				if !covered {
					continue
				}

				nisa, bad := yenAt(balanceSlot, row, nisaColumn, fields[atBalance[nisaColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				taxable, bad := yenAt(balanceSlot, row, taxableColumn, fields[atBalance[taxableColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				basis, bad := yenAt(balanceSlot, row, basisColumn, fields[atBalance[basisColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}

				rest := outside[year]
				missingPot := false
				for _, pocket := range []string{nisaPocket, taxablePocket} {
					if _, written := rest[pocket]; !written {
						found = append(found, Finding{
							Slot: outsideSlot,
							Message: fmt.Sprintf("%s 年の %s の行が無い。報告書は夫の投資信託しか載せていないので、残りは差額ではなく書き留めた数でなければならない。外に何も無いなら 0 と書く",
								year, pocket),
						})
						missingPot = true
					}
				}
				if missingPot {
					continue
				}

				if want := statement[nisaPocket].value + rest[nisaPocket]; nisa != want {
					found = append(found, Finding{
						Slot: balanceSlot,
						Message: fmt.Sprintf("%s 年の %s が %d 円。報告書の %d 円に、報告書の外の %d 円を足した %d 円であるべき",
							year, nisaColumn, nisa, statement[nisaPocket].value, rest[nisaPocket], want),
					})
				}

				held := statement[taxablePocket]
				if want := held.value + rest[taxablePocket]; taxable != want {
					found = append(found, Finding{
						Slot: balanceSlot,
						Message: fmt.Sprintf("%s 年の %s が %d 円。報告書の %d 円に、報告書の外の %d 円を足した %d 円であるべき",
							year, taxableColumn, taxable, held.value, rest[taxablePocket], want),
					})
				}
				if want := held.basis + rest[taxablePocket]; basis != want {
					found = append(found, Finding{
						Slot: balanceSlot,
						Message: fmt.Sprintf(
							"%s 年の %s が %d 円。報告書の取得価額 %d 円に、報告書の外の %d 円を含み益ゼロで足した %d 円であるべき",
							year, basisColumn, basis, held.basis, rest[taxablePocket], want),
					})
				}
				if basis > taxable {
					found = append(found, Finding{
						Slot: balanceSlot,
						Message: fmt.Sprintf("%s 年の %s が %d 円で、%s の %d 円を超えている",
							year, basisColumn, basis, taxableColumn, taxable),
					})
				}
			}
			return found
		},
	}
}

func yearEnding(date string) (string, bool) {
	if len(date) != len("2025-12-31") || date[len(date)-len("-12-31"):] != "-12-31" {
		return "", false
	}
	return date[:4], true
}

type SplitBalanceSide struct {
	Slot tsv.Slot

	Year, NISA, Taxable, Basis tsv.ColumnName
}

type StatementSide struct {
	Slot tsv.Slot

	AsOf, Pocket, Value, Gain tsv.ColumnName
}

type OutsideSide struct {
	Slot tsv.Slot

	Year, Amount, Pocket tsv.ColumnName
}
