package validate

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
)

const KeysCoverYearsRule = "keys-cover-years"

func KeysCoverYears(partsSlot tsv.Slot, partsYearColumn, keyColumn tsv.ColumnName, wholeSlot tsv.Slot, wholeYearColumn tsv.ColumnName) Rule {
	needs := []tsv.Slot{partsSlot, wholeSlot}
	slices.Sort(needs)

	return Rule{
		Name:  KeysCoverYearsRule,
		Needs: needs,
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			parts := tables[partsSlot]
			whole := tables[wholeSlot]

			columns, problems := columnsOf(parts, partsSlot, partsYearColumn, keyColumn)
			if len(problems) > 0 {
				return problems
			}
			wholeColumns, problems := columnsOf(whole, wholeSlot, wholeYearColumn)
			if len(problems) > 0 {
				return problems
			}

			var keys []string
			named := make(map[string]map[string]bool, 8)
			for _, fields := range parts.Rows {
				year, key := fields[columns[partsYearColumn]], fields[columns[keyColumn]]
				if !slices.Contains(keys, key) {
					keys = append(keys, key)
				}
				if named[year] == nil {
					named[year] = make(map[string]bool, 8)
				}
				named[year][key] = true
			}
			slices.Sort(keys)

			var found []Finding
			for row, fields := range whole.Rows {
				year := fields[wholeColumns[wholeYearColumn]]

				var absent []string
				for _, key := range keys {
					if !named[year][key] {
						absent = append(absent, key)
					}
				}
				if len(absent) == 0 {
					continue
				}
				found = append(found, Finding{
					Slot: partsSlot,
					Message: fmt.Sprintf(
						"row %d of %q says %s, but %s names no %v for it (a missing row is not a nought)",
						row+1, wholeSlot, year, partsSlot, absent),
				})
			}
			return found
		},
	}
}
