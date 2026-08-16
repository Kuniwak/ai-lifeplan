package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	LawRangeTotalRule RuleName = "law-range-total"
	LawSourceRule     RuleName = "law-source"
	LawValidityRule   RuleName = "law-validity"
	MunicipalityRule  RuleName = "municipality-supported"

	LastMunicipalityRule RuleName = "last-municipality-supported"
)

type LawYearWord string

const Indefinite LawYearWord = "無期限"

const Unknown LawYearWord = "不明"

func LawRangeTotal(slot tsv.Slot, lowerColumn tsv.ColumnName, domainMin int64) Rule {
	return Rule{
		Name:  LawRangeTotalRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			i, ok := table.ColumnIndex(lowerColumn)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("the column %q is missing", lowerColumn),
				}}
			}

			if len(table.Rows) == 0 {
				return []Finding{{
					Slot:    slot,
					Message: "has no bands, so every lookup into it would miss",
				}}
			}

			var found []Finding
			bounds := make([]int64, 0, len(table.Rows))
			for row, fields := range table.Rows {
				bound, err := money.ParseYen(fields[i])
				if err != nil {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("row %d, column %q: %q is not a bound", row+1, lowerColumn, fields[i]),
					})
					continue
				}
				bounds = append(bounds, int64(bound))
			}
			if len(found) > 0 {
				return found
			}

			slices.Sort(bounds)

			if lowest := bounds[0]; lowest > domainMin {
				found = append(found, Finding{
					Slot: slot,
					Message: fmt.Sprintf(
						"the lowest band starts at %d, leaving %d..%d uncovered, so a lookup below %d would miss",
						lowest, domainMin, lowest-1, lowest),
				})
			}

			for j := 1; j < len(bounds); j++ {
				if bounds[j] == bounds[j-1] {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("two bands start at %d, so neither can be said to apply there", bounds[j]),
					})
				}
			}

			return found
		},
	}
}

func LawSource(slot tsv.Slot, sourceColumn tsv.ColumnName) Rule {
	return Rule{
		Name:  LawSourceRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			i, ok := table.ColumnIndex(sourceColumn)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("the column %q is missing", sourceColumn),
				}}
			}

			var found []Finding
			for row, fields := range table.Rows {
				if strings.TrimSpace(fields[i]) == "" {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("row %d has no source, so the figure cannot be traced to what set it", row+1),
					})
				}
			}
			return found
		},
	}
}

func LawValidity(slot tsv.Slot, startColumn, endColumn tsv.ColumnName) Rule {
	return Rule{
		Name:  LawValidityRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			startIndex, ok := table.ColumnIndex(startColumn)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("the column %q is missing", startColumn),
				}}
			}
			endIndex, ok := table.ColumnIndex(endColumn)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("the column %q is missing", endColumn),
				}}
			}

			var found []Finding
			for row, fields := range table.Rows {
				var start date.Year
				known := LawYearWord(fields[startIndex]) != Unknown
				if known {
					parsed, err := date.ParseYear(fields[startIndex])
					if err != nil {
						message := fmt.Sprintf("row %d, column %q: %q is not a year", row+1, startColumn, fields[startIndex])
						if strings.TrimSpace(fields[startIndex]) == "" {
							message = fmt.Sprintf(
								"row %d has a blank %s; write %q when the commencement has not been looked up, so that it is not confused with an oversight",
								row+1, startColumn, Unknown)
						}
						found = append(found, Finding{Slot: slot, Message: message})
						continue
					}
					start = parsed
				}

				end := fields[endIndex]
				if end == "" {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"row %d has a blank %s; write %q when no end has been announced, so that it is not confused with an end nobody looked up",
							row+1, endColumn, Indefinite),
					})
					continue
				}
				if LawYearWord(end) == Indefinite {
					continue
				}

				endYear, err := date.ParseYear(end)
				if err != nil {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("row %d, column %q: %q is neither a year nor %q",
							row+1, endColumn, end, Indefinite),
					})
					continue
				}
				if known && endYear < start {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("row %d ends in %d, before it starts in %d", row+1, endYear, start),
					})
				}
			}
			return found
		},
	}
}

type MunicipalityGate struct {
	Rule RuleName

	What string

	LastOnly bool

	Supported []string

	Missing func(string) []string
}

func MunicipalitySupported(slot tsv.Slot, municipalityColumn tsv.ColumnName, gate MunicipalityGate) Rule {
	return Rule{
		Name:  gate.Rule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			i, ok := table.ColumnIndex(municipalityColumn)
			if !ok {
				return []Finding{{
					Slot:    slot,
					Message: fmt.Sprintf("the column %q is missing", municipalityColumn),
				}}
			}

			var asked []string
			for _, fields := range table.Rows {
				asked = append(asked, fields[i])
			}
			if gate.LastOnly {
				if len(asked) == 0 {
					return nil
				}
				asked = asked[len(asked)-1:]
			}
			slices.Sort(asked)
			asked = slices.Compact(asked)

			var found []Finding
			for _, name := range asked {
				if name == "" {
					continue
				}
				if !slices.Contains(gate.Supported, name) {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"the plan's %s is %q, which is not written up in full; add %s rather than falling back to the national figures",
							gate.What, name, strings.Join(gate.Missing(name), ", ")),
					})
				}
			}
			return found
		},
	}
}
