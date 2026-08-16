package validate

import (
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
)

const YearsAreCoveredRule RuleName = "years-are-covered"

var yearInName = regexp.MustCompile(`(^|[^0-9])([12][0-9]{3})([^0-9]|$)`)

func YearsAreCovered(
	sourceSlot tsv.Slot, fileColumn tsv.ColumnName,
	tableSlot tsv.Slot, monthColumn tsv.ColumnName,
) Rule {
	return Rule{
		Name:  YearsAreCoveredRule,
		Needs: []tsv.Slot{sourceSlot, tableSlot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			sources, produced := tables[sourceSlot], tables[tableSlot]

			atSource, found := columnsOf(sources, sourceSlot, fileColumn)
			atProduced, missing := columnsOf(produced, tableSlot, monthColumn)
			found = append(found, missing...)
			if len(found) > 0 {
				return found
			}

			read := make(map[string]bool, len(sources.Rows))
			for row, fields := range sources.Rows {
				name := fields[atSource[fileColumn]]
				match := yearInName.FindStringSubmatch(name)
				if match == nil {
					found = append(found, Finding{
						Slot: sourceSlot,
						Message: fmt.Sprintf("row %d: %q から年が読めない。どの年を読んだ書き出しなのかが分からないと、その年が落ちても言えない",
							row+1, name),
					})
					continue
				}
				read[match[2]] = true
			}
			if len(found) > 0 {
				return found
			}

			has := make(map[string]bool, len(produced.Rows))
			for row, fields := range produced.Rows {
				month := fields[atProduced[monthColumn]]
				if len(month) < 4 {
					found = append(found, Finding{
						Slot: tableSlot,
						Message: fmt.Sprintf("row %d: %q から年が読めない。どの年の行なのかが分からないと、その年が落ちても言えない",
							row+1, month),
					})
					continue
				}
				has[month[:4]] = true
			}
			if len(found) > 0 {
				return found
			}

			for _, year := range slices.Sorted(maps.Keys(read)) {
				if !has[year] {
					found = append(found, Finding{
						Slot: tableSlot,
						Message: fmt.Sprintf("%s 年の書き出しを読んだと %s が言っているのに、%s にその年の行が 1 つも無い",
							year, sourceSlot, tableSlot),
					})
				}
			}
			for _, year := range slices.Sorted(maps.Keys(has)) {
				if !read[year] {
					found = append(found, Finding{
						Slot: sourceSlot,
						Message: fmt.Sprintf("%s に %s 年の行があるのに、その年を読んだ書き出しが %s に無い",
							tableSlot, year, sourceSlot),
					})
				}
			}
			return found
		},
	}
}
