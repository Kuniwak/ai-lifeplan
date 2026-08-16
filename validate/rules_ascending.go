package validate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const AscendingRule = "ascending"

func Ascending(slot tsv.Slot, column tsv.ColumnName) Rule {
	return Rule{
		Name:  AscendingRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			i, ok := table.ColumnIndex(column)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("%q という列が無い。並び順を確かめられない", column),
				}}
			}

			var found []Finding
			numbers := make([]int, 0, len(table.Rows))
			for row, fields := range table.Rows {
				n, err := strconv.Atoi(strings.TrimSpace(fields[i]))
				if err != nil {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("%d 行目: %s が %q で、整数ではない", row+1, column, fields[i]),
					})
					continue
				}
				numbers = append(numbers, n)
			}

			for i := 1; i < len(numbers); i++ {
				switch {
				case numbers[i] == numbers[i-1]:
					found = append(found, Finding{
						Slot: slot,
						Message: wording.DuplicateKeyFinding(string(column), wording.Number(numbers[i]),
							wording.WhichRowDecidesTheValue),
					})
				case numbers[i] < numbers[i-1]:
					found = append(found, Finding{
						Slot: slot,
						Message: wording.OutOfAscendingOrderFinding(string(column),
							wording.Number(numbers[i-1]), wording.Number(numbers[i])),
					})
				}
			}

			return found
		},
	}
}
