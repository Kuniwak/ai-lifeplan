package validate

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/stepfn"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const (
	ColumnSchemaRule    RuleName = "column-schema"
	YearGapRule         RuleName = "year-gap"
	StepMonotonicRule   RuleName = "step-monotonic"
	StepCoversStartRule RuleName = "step-covers-start"
	ValueRangeRule      RuleName = "value-range"
)

func ColumnSchema(slot tsv.Slot, columns []Column) Rule {
	return Rule{
		Name:  ColumnSchemaRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]
			var found []Finding

			for _, column := range columns {
				i, ok := table.ColumnIndex(column.Name)
				if !ok {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("the column %q is missing", column.Name),
					})
					continue
				}

				for row, fields := range table.Rows {
					if err := column.Parse(fields[i]); err != nil {
						found = append(found, unreadableField(slot, row, column.Name, err))
					}
				}
			}

			return found
		},
	}
}

func YearGap(slot tsv.Slot, yearColumn tsv.ColumnName, from, to date.Year) Rule {
	return Rule{
		Name:  YearGapRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			years, found := readYears(tables[slot], slot, yearColumn)
			if len(found) > 0 {
				return found
			}

			seen := make(map[date.Year]int, len(years))
			for _, y := range years {
				seen[y]++
			}

			for y := from; y <= to; y++ {
				switch seen[y] {
				case 1:
				case 0:
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("year %d is missing", y),
					})
				case 2:
					found = append(found, Finding{
						Slot: slot,
						Message: wording.DuplicateKeyFinding(string(yearColumn), wording.Number(y),
							wording.WhichRowReadsTheYear),
					})
				default:
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("%s。%d 度書かれている",
							wording.DuplicateKeyFinding(string(yearColumn), wording.Number(y), wording.WhichRowReadsTheYear),
							seen[y]),
					})
				}
			}

			for _, y := range years {
				if y < from || y > to {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("year %d is outside the span %d..%d of the plan", y, from, to),
					})
				}
			}

			return found
		},
	}
}

func StepMonotonic(slot tsv.Slot, yearColumn tsv.ColumnName) Rule {
	return Rule{
		Name:  StepMonotonicRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			years, found := readYears(tables[slot], slot, yearColumn)
			if len(found) > 0 {
				return found
			}

			for i := 1; i < len(years); i++ {
				if err := stepfn.OutOfOrder(years[i-1], years[i]); err != nil {
					found = append(found, Finding{Slot: slot, Message: err.Error()})
				}
			}

			return found
		},
	}
}

func StepCoversStart(slot tsv.Slot, yearColumn tsv.ColumnName, planStart date.Year) Rule {
	return Rule{
		Name:  StepCoversStartRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			years, found := readYears(tables[slot], slot, yearColumn)
			if len(found) > 0 {
				return found
			}

			if err := stepfn.NoValueYet(planStart, years); err != nil {
				return []Finding{{Slot: slot, Message: err.Error()}}
			}
			return nil
		},
	}
}

func ValueRange(slot tsv.Slot, column tsv.ColumnName, minimum, maximum int64) Rule {
	return Rule{
		Name:  ValueRangeRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			i, ok := table.ColumnIndex(column)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("the column %q is missing", column),
				}}
			}

			var found []Finding
			for row, fields := range table.Rows {
				amount, err := money.ParseYen(fields[i])
				if err != nil {
					continue
				}
				if value := int64(amount); value < minimum || value > maximum {
					found = append(found, fieldFinding(slot, row, column,
						fmt.Sprintf("%d is outside %d..%d", value, minimum, maximum)))
				}
			}
			return found
		},
	}
}

func readYears(table *tsv.Table, slot tsv.Slot, yearColumn tsv.ColumnName) ([]date.Year, []Finding) {
	i, ok := table.ColumnIndex(yearColumn)
	if !ok {
		return nil, []Finding{{
			Slot:    slot,
			Message: fmt.Sprintf("the year column %q is missing", yearColumn),
		}}
	}

	years := make([]date.Year, 0, len(table.Rows))
	var found []Finding
	for row, fields := range table.Rows {
		y, err := date.ParseYear(fields[i])
		if err != nil {
			found = append(found, fieldFinding(slot, row, yearColumn,
				fmt.Sprintf("%q is not a year", fields[i])))
			continue
		}
		years = append(years, y)
	}

	return years, found
}
