package plan_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/yeartest"
)

func firstPensionRow(t *testing.T, manifest string) map[tsv.ColumnName]string {
	t.Helper()

	written, ok := theProject(t, manifest).Tables()["income-husband"]
	if !ok {
		t.Fatalf("%s の書き出しに income-husband が無い", manifest)
	}

	var found map[tsv.ColumnName]string
	yeartest.EachYear(t, written, func(_ date.Year, fields []string) {
		if found != nil {
			return
		}
		row := make(map[tsv.ColumnName]string, len(written.Header))
		for at, name := range written.Header {
			row[name] = fields[at]
		}
		if row["年金収入"] != "0" && row["年金収入"] != "" {
			found = row
		}
	})
	if found == nil {
		t.Fatalf("%s に年金を受け取る年が無い", manifest)
	}
	return found
}
