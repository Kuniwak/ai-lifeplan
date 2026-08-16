package validate

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	UnclassifiedRule RuleName = "actuals-unclassified"
	AmountsAddUpRule RuleName = "actuals-amounts-add-up"
	ItemsKnownRule   RuleName = "actuals-items-known"
)

func Unclassified(
	slot tsv.Slot, itemColumn, amountColumn tsv.ColumnName, unclassified string,
	limit money.Rate, groupColumn tsv.ColumnName, groupPrefix int,
) Rule {
	return Rule{
		Name:  UnclassifiedRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, itemColumn, amountColumn, groupColumn)
			if len(found) > 0 {
				return found
			}

			spent := make(map[string]money.Yen)
			unknown := make(map[string]money.Yen)
			for row, fields := range table.Rows {
				amount, err := money.ParseYen(fields[at[amountColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot, Message: fmt.Sprintf("row %d: %v", row+1, err)})
					continue
				}
				if amount >= 0 {
					continue
				}

				group := fields[at[groupColumn]]
				if groupPrefix > 0 && len(group) > groupPrefix {
					group = group[:groupPrefix]
				}
				spent[group] -= amount
				if fields[at[itemColumn]] == unclassified {
					unknown[group] -= amount
				}
			}

			groups := make([]string, 0, len(spent))
			for group := range spent {
				groups = append(groups, group)
			}
			slices.Sort(groups)

			for _, group := range groups {
				if spent[group] == 0 {
					continue
				}
				if unknown[group]*money.Yen(limit.Den()) > spent[group]*money.Yen(limit.Num()) {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"%s: %d of %d spent is %q, which is over the %s this has been held to",
							group, unknown[group], spent[group], unclassified, limit),
					})
				}
			}
			return found
		},
	}
}

func AmountsAddUp(slot tsv.Slot, totalColumn tsv.ColumnName, partColumns []tsv.ColumnName) Rule {
	return Rule{
		Name:  AmountsAddUpRule + RuleName(ScopeSeparator+totalColumn),
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, append([]tsv.ColumnName{totalColumn}, partColumns...)...)
			if len(found) > 0 {
				return found
			}

			for row, fields := range table.Rows {
				total, err := money.ParseYen(fields[at[totalColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot, Message: fmt.Sprintf("row %d: %v", row+1, err)})
					continue
				}

				var parts money.Yen
				for _, column := range partColumns {
					part, err := money.ParseYen(fields[at[column]])
					if err != nil {
						found = append(found, Finding{Slot: slot, Message: fmt.Sprintf("row %d: %v", row+1, err)})
						continue
					}
					parts += part
				}

				if parts != total {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("row %d: %q is %d but its parts come to %d",
							row+1, totalColumn, total, parts),
					})
				}
			}
			return found
		},
	}
}

func ItemsKnown(slot tsv.Slot, itemColumn tsv.ColumnName, known []string) Rule {
	allowed := slices.Clone(known)
	slices.Sort(allowed)

	return Rule{
		Name:  ItemsKnownRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, itemColumn)
			if len(found) > 0 {
				return found
			}

			reported := make(map[string]bool)
			for row, fields := range table.Rows {
				item := fields[at[itemColumn]]
				if _, ok := slices.BinarySearch(allowed, item); ok || reported[item] {
					continue
				}
				reported[item] = true
				found = append(found, Finding{
					Slot:    slot,
					Message: fmt.Sprintf("row %d: %q is not an item the plan has a place for; the ones it has are %v", row+1, item, allowed),
				})
			}
			return found
		},
	}
}

func columnsOf(table *tsv.Table, slot tsv.Slot, columns ...tsv.ColumnName) (map[tsv.ColumnName]int, []Finding) {
	at := make(map[tsv.ColumnName]int, len(columns))
	var found []Finding
	for _, column := range columns {
		i, ok := table.ColumnIndex(column)
		if !ok {
			found = append(found, Finding{Slot: slot, Message: fmt.Sprintf("the column %q is missing", column)})
			continue
		}
		at[column] = i
	}
	return at, found
}
