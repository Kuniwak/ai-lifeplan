package input

import (
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
)

func UnreadColumns(slot tsv.Slot, table *tsv.Table) []tsv.ColumnName {
	i := slices.IndexFunc(shapes, func(s Shape) bool { return s.Slot == slot })
	if i < 0 {
		return nil
	}

	described := shapes[i].Columns
	declared := make(map[tsv.ColumnName]struct{}, len(described))
	for _, column := range described {
		declared[column.Name] = struct{}{}
	}

	var unread []tsv.ColumnName
	for _, column := range table.Header {
		if _, described := declared[column]; !described {
			unread = append(unread, column)
		}
	}
	slices.Sort(unread)
	return unread
}

func UnreadColumnsOf(tables map[tsv.Slot]*tsv.Table) map[tsv.Slot][]tsv.ColumnName {
	unread := make(map[tsv.Slot][]tsv.ColumnName, len(tables))
	for slot, table := range tables {
		if columns := UnreadColumns(slot, table); len(columns) > 0 {
			unread[slot] = columns
		}
	}
	return unread
}
