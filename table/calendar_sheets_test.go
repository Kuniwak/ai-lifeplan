package table_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/yeartest"
)

const (
	repoRoot        = ".."
	classicManifest = "../projects/classic.tsv"
)

func tablesOfTheBaseProject(t *testing.T) map[tsv.Slot]*tsv.Table {
	t.Helper()

	p, err := project.Load(classicManifest)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}
	paths := make(map[tsv.Slot]string, len(p.SlotNames()))
	for _, slot := range p.SlotNames() {
		path, _ := p.Path(slot)
		paths[slot] = path
	}
	tables, err := input.Load(repoRoot, paths)
	if err != nil {
		t.Fatalf("input.Load: %v", err)
	}
	return tables
}

func calendarOfTheBaseProject(t *testing.T) relation.Table[table.CalendarRow] {
	t.Helper()

	in, err := table.CalendarInputFrom(tablesOfTheBaseProject(t))
	if err != nil {
		t.Fatalf("table.CalendarInputFrom: %v", err)
	}
	built, err := table.Calendar(in)
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}
	return built
}

func read(t *testing.T, path string) *tsv.Table {
	t.Helper()

	table, err := tsv.ReadFile(path)
	if err != nil {
		t.Fatalf("tsv.ReadFile: %v", err)
	}
	return table
}

func columnIndex(t *testing.T, table *tsv.Table, name tsv.ColumnName) int {
	t.Helper()
	return yeartest.ColumnIndex(t, table, name)
}

func forEachSheetsYear(t *testing.T, golden *tsv.Table, f func(year date.Year, fields []string)) {
	t.Helper()
	yeartest.EachSheetsYear(t, golden, f)
}

func builtRowAt[T any](t *testing.T, built relation.Table[T], year date.Year) T {
	t.Helper()
	return yeartest.RowAt(t, built, year)
}

func number(t *testing.T, table *tsv.Table, fields []string, column tsv.ColumnName, unit string) int {
	t.Helper()

	field := strings.TrimSuffix(fields[columnIndex(t, table, column)], unit)
	n, err := strconv.Atoi(field)
	if err != nil {
		t.Fatalf("column %q: %v", column, err)
	}
	return n
}
