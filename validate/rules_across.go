package validate

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/sets"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	YearCoverageRule   RuleName = "year-coverage"
	SlotResolvableRule RuleName = "slot-resolvable"
)

func YearCoverage(slots []tsv.Slot, yearColumn tsv.ColumnName, from, to date.Year) Rule {
	needs := slices.Clone(slots)
	slices.Sort(needs)

	return Rule{
		Name:  YearCoverageRule,
		Needs: needs,
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			want := relation.Span(from, to)

			var found []Finding
			for _, slot := range needs {
				years, problems := readYears(tables[slot], slot, yearColumn)
				if len(problems) > 0 {
					found = append(found, problems...)
					continue
				}

				slices.Sort(years)
				years = slices.Compact(years)

				if absent := sets.Difference(want, years); len(absent) > 0 {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"does not cover %d year(s) that the plan spans: %v",
							len(absent), absent),
					})
				}
				if extra := sets.Difference(years, want); len(extra) > 0 {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"covers %d year(s) outside the span %d..%d of the plan: %v",
							len(extra), from, to, extra),
					})
				}
			}

			return found
		},
	}
}

type Exists func(path string) bool

func SlotResolvable(required []tsv.Slot, paths map[tsv.Slot]string, exists Exists) Rule {
	return Rule{
		Name:  SlotResolvableRule,
		Needs: []tsv.Slot{},
		Check: func(map[tsv.Slot]*tsv.Table) []Finding {
			wanted := slices.Clone(required)
			slices.Sort(wanted)

			var found []Finding
			for _, slot := range wanted {
				path, set := paths[slot]
				if !set {
					found = append(found, Finding{
						Slot:    slot,
						Message: "no layer sets this slot, so the plan has no table for it",
					})
					continue
				}
				if !exists(path) {
					found = append(found, Finding{
						Slot:    slot,
						Message: fmt.Sprintf("points at %q, which is not there", path),
					})
				}
			}
			return found
		},
	}
}
