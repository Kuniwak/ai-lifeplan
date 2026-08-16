package validate

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/sets"
	"github.com/Kuniwak/lifeplan/tsv"
)

const YearsOutsideComparisonRule = "years-outside-comparison"

type Comparable int

const (
	AgainstMovement Comparable = iota

	AgainstLevel
)

func YearsOutsideComparison(flowsSlot tsv.Slot, monthColumn tsv.ColumnName, balancesSlot tsv.Slot, yearColumn tsv.ColumnName, against Comparable, expected []string) Rule {
	needs := []tsv.Slot{flowsSlot, balancesSlot}
	slices.Sort(needs)

	return Rule{
		Name:  YearsOutsideComparisonRule,
		Needs: needs,
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			flows, balances := tables[flowsSlot], tables[balancesSlot]

			flowColumns, problems := columnsOf(flows, flowsSlot, monthColumn)
			if len(problems) > 0 {
				return problems
			}
			balanceColumns, problems := columnsOf(balances, balancesSlot, yearColumn)
			if len(problems) > 0 {
				return problems
			}

			const yearPrefix = 4

			var reached []string
			for _, fields := range flows.Rows {
				month := fields[flowColumns[monthColumn]]
				if len(month) < yearPrefix {
					return []Finding{{
						Slot:    flowsSlot,
						Message: fmt.Sprintf("%q is not a 年月; it has no year to read", month),
					}}
				}
				reached = append(reached, month[:yearPrefix])
			}
			slices.Sort(reached)
			reached = slices.Compact(reached)

			var held []string
			for _, fields := range balances.Rows {
				held = append(held, fields[balanceColumns[yearColumn]])
			}
			slices.Sort(held)
			held = slices.Compact(held)

			comparable := held
			if against == AgainstMovement && len(comparable) > 0 {
				comparable = comparable[1:]
			}

			outside := sets.Difference(reached, comparable)
			want := slices.Clone(expected)
			slices.Sort(want)

			var found []Finding
			if joined := sets.Difference(outside, want); len(joined) > 0 {
				found = append(found, Finding{
					Slot: flowsSlot,
					Message: fmt.Sprintf(
						"%v は突合の外にあるが、そう書かれていない。この年の収支明細は誰にも読まれない",
						joined),
				})
			}
			if left := sets.Difference(want, outside); len(left) > 0 {
				found = append(found, Finding{
					Slot: flowsSlot,
					Message: fmt.Sprintf(
						"%v は突合の外だと書かれているが、いまは突合できている。書き換えないと、次に外へ落ちた年がこれに隠れる",
						left),
				})
			}
			return found
		},
	}
}
