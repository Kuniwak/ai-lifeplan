package validate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

const PositiveNeedsRule RuleName = "positive-needs"

type PositivePair struct {
	Positive, Needed tsv.ColumnName

	Why string
}

func PositiveNeeds(slot tsv.Slot, yearColumn tsv.ColumnName, needs []PositivePair) Rule {
	return Rule{
		Name:  PositiveNeedsRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			columns := []tsv.ColumnName{yearColumn}
			for _, need := range needs {
				columns = append(columns, need.Positive, need.Needed)
			}
			at, found := columnsOf(table, slot, columns...)
			if len(found) > 0 {
				return found
			}

			for row, fields := range table.Rows {
				year := fields[at[yearColumn]]

				for _, need := range needs {
					positive, err := SignOf(fields[at[need.Positive]])
					if err != nil {
						found = append(found, Finding{
							Slot:    slot,
							Message: fmt.Sprintf("row %d (%s): %s: %v", row+1, year, need.Positive, err),
						})
						continue
					}
					if positive <= 0 {
						continue
					}

					needed, err := SignOf(fields[at[need.Needed]])
					if err != nil {
						found = append(found, Finding{
							Slot:    slot,
							Message: fmt.Sprintf("row %d (%s): %s: %v", row+1, year, need.Needed, err),
						})
						continue
					}
					if needed <= 0 {
						found = append(found, Finding{
							Slot: slot,
							Message: fmt.Sprintf(
								"row %d (%s): %s が %d なのに %s が %d である。%s",
								row+1, year, need.Positive, positive, need.Needed, needed, need.Why),
						})
					}
				}
			}
			return found
		},
	}
}

func SignOf(field string) (int64, error) {
	n, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(field), ",", ""), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("数でない: %q", field)
	}
	return n, nil
}
