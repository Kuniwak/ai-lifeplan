package plan_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/yeartest"
)

func columnIndexOf(t *testing.T, table *tsv.Table, name tsv.ColumnName) int {
	t.Helper()
	return yeartest.ColumnIndex(t, table, name)
}

const childAllowanceLimitsFrom date.Year = 2022
