package validate

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const OutsideFollowsTheHoldingsRule RuleName = "outside-follows-holdings"

func OutsideFollowsTheHoldings(
	outsideSlot tsv.Slot, outsideYearColumn, outsideAmountColumn, outsidePocketColumn tsv.ColumnName,
	holdingsSlot tsv.Slot, asOfColumn, holdingsPocketColumn, valueColumn tsv.ColumnName,
) Rule {
	return Rule{
		Name:  OutsideFollowsTheHoldingsRule,
		Needs: []tsv.Slot{outsideSlot, holdingsSlot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			rest, found := columnsOf(tables[outsideSlot], outsideSlot,
				outsideYearColumn, outsideAmountColumn, outsidePocketColumn)
			holds, missing := columnsOf(tables[holdingsSlot], holdingsSlot,
				asOfColumn, holdingsPocketColumn, valueColumn)
			found = append(found, missing...)
			if len(found) > 0 {
				return found
			}

			last := make(map[string]string)
			for _, fields := range tables[holdingsSlot].Rows {
				asOf := fields[holds[asOfColumn]]
				if len(asOf) < 4 {
					return []Finding{{
						Slot:    holdingsSlot,
						Message: fmt.Sprintf("%q は基準日として読めない", asOf),
					}}
				}
				if year := asOf[:4]; asOf > last[year] {
					last[year] = asOf
				}
			}

			held := make(map[string]map[string]money.Yen, len(last))
			for row, fields := range tables[holdingsSlot].Rows {
				asOf := fields[holds[asOfColumn]]
				year := asOf[:4]
				if asOf != last[year] {
					continue
				}
				if !strings.HasPrefix(asOf[4:], "-12-") {
					found = append(found, Finding{
						Slot: holdingsSlot,
						Message: fmt.Sprintf(
							"%s 年の最後の基準日が %s である。年末ではないものを年末として読むわけにはいかない",
							year, asOf),
					})
					continue
				}
				value, bad := yenAt(holdingsSlot, row, valueColumn, fields[holds[valueColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				if _, seen := held[year]; !seen {
					held[year] = make(map[string]money.Yen, 2)
				}
				held[year][fields[holds[holdingsPocketColumn]]] += value
			}
			if len(found) > 0 {
				return found
			}

			written := make(map[string]map[string]money.Yen, len(held))
			for row, fields := range tables[outsideSlot].Rows {
				amount, bad := yenAt(outsideSlot, row, outsideAmountColumn, fields[rest[outsideAmountColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}
				year := fields[rest[outsideYearColumn]]
				if _, seen := written[year]; !seen {
					written[year] = make(map[string]money.Yen, 2)
				}
				written[year][fields[rest[outsidePocketColumn]]] += amount
			}
			if len(found) > 0 {
				return found
			}

			for _, year := range slices.Sorted(maps.Keys(held)) {
				for _, pocket := range slices.Sorted(maps.Keys(held[year])) {
					want := held[year][pocket]
					got, ok := written[year][pocket]
					if !ok {
						found = append(found, Finding{
							Slot: outsideSlot,
							Message: fmt.Sprintf("%s 年の %s の行が無い。持ち高の表は %d 円を持っている",
								year, pocket, want),
						})
						continue
					}
					if got != want {
						found = append(found, boundColumnFinding(
							outsideSlot, year, tsv.ColumnName(pocket), got, "持ち高の表の年末", want))
					}
				}
			}
			return found
		},
	}
}
