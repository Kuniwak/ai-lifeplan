package yeartest

import (
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const YearColumn tsv.ColumnName = "西暦"

type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

func EachYear(t TB, table *tsv.Table, f func(year date.Year, fields []string)) {
	t.Helper()
	eachYear(t, table, YearColumn, false, f)
}

func EachYearOf(t TB, table *tsv.Table, column tsv.ColumnName, f func(year date.Year, fields []string)) {
	t.Helper()
	eachYear(t, table, column, false, f)
}

func EachSheetsYear(t TB, golden *tsv.Table, f func(year date.Year, fields []string)) {
	t.Helper()
	eachYear(t, golden, YearColumn, true, f)
}

func eachYear(t TB, table *tsv.Table, column tsv.ColumnName, sheets bool, f func(year date.Year, fields []string)) {
	t.Helper()

	at, ok := table.ColumnIndex(column)
	if !ok {
		t.Fatalf("no %q column; the table has %v", column, table.Header)
		return
	}
	for row, fields := range table.Rows {
		field := fields[at]
		if sheets {
			field = strings.TrimSuffix(strings.TrimSpace(field), "年")
		}
		year, err := date.ParseYear(field)
		if err != nil {
			t.Fatalf("row %d, column %q: %v", row+1, column, err)
			return
		}
		f(year, fields)
	}
}

func RowAt[T any](t TB, built relation.Table[T], year date.Year) T {
	t.Helper()

	row, ok := built.At(year)
	if !ok {
		t.Fatalf("%d is missing from what the program built", year)
	}
	return row
}

func ColumnIndex(t TB, table *tsv.Table, name tsv.ColumnName) int {
	t.Helper()

	at, ok := table.ColumnIndex(name)
	if !ok {
		t.Fatalf("no %q column; the table has %v", name, table.Header)
		return 0
	}
	return at
}
