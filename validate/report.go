package validate

import (
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	RuleColumn    tsv.ColumnName = "rule"
	NeedsColumn   tsv.ColumnName = "needs"
	StatusColumn  tsv.ColumnName = "status"
	SlotColumn    tsv.ColumnName = "slot"
	MessageColumn tsv.ColumnName = "message"
)

type Status string

const (
	StatusRan     Status = "ran"
	StatusSkipped Status = "skipped"
)

func ListTable(registry Registry) *tsv.Table {
	table := &tsv.Table{Header: []tsv.ColumnName{RuleColumn, NeedsColumn}}
	for _, rule := range registry.Rules() {
		table.Rows = append(table.Rows, []string{string(rule.Name), joinSlots(rule.Needs, " ")})
	}
	return table
}

func CoverageTable(result Result) *tsv.Table {
	table := &tsv.Table{Header: []tsv.ColumnName{RuleColumn, StatusColumn}}
	for _, name := range result.Ran {
		table.Rows = append(table.Rows, []string{string(name), string(StatusRan)})
	}
	for _, name := range result.Skipped {
		table.Rows = append(table.Rows, []string{string(name), string(StatusSkipped)})
	}
	return table
}

func FindingsTable(result Result) *tsv.Table {
	table := &tsv.Table{Header: []tsv.ColumnName{RuleColumn, SlotColumn, MessageColumn}}
	for _, f := range result.Findings {
		table.Rows = append(table.Rows, []string{string(f.Rule), string(f.Slot), f.Message})
	}
	return table
}

func joinSlots(slots []tsv.Slot, separator string) string {
	names := make([]string, 0, len(slots))
	for _, slot := range slots {
		names = append(names, string(slot))
	}
	return strings.Join(names, separator)
}
