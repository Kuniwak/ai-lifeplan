package compare_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func record(t *testing.T, rows ...[]string) actuals.BalanceTable {
	t.Helper()

	written := make([][]string, len(rows))
	for i, row := range rows {
		year, cash, invested, locked, partial := row[0], row[1], row[2], row[3], row[4]
		written[i] = []string{year, cash, invested, "0", invested, locked, partial}
	}

	parsed, err := actuals.ParseBalanceTable(&tsv.Table{
		Header: []tsv.ColumnName{
			actuals.BalanceYearColumn, actuals.BalanceCashColumn,
			actuals.BalanceInvestedColumn, actuals.BalanceNISAColumn,
			actuals.BalanceTaxableColumn, actuals.BalanceLockedColumn,
			actuals.BalancePartialColumn,
		},
		Rows: written,
	})
	if err != nil {
		t.Fatalf("ParseBalanceTable: %v", err)
	}
	return parsed
}

func tables(name plan.TableName, header []tsv.ColumnName, rows ...[]string) map[plan.TableName]*tsv.Table {
	return map[plan.TableName]*tsv.Table{name: {Header: header, Rows: rows}}
}

func twoYears(receipts, spending, balance, total, shortfall, tax [2]string) map[plan.TableName]*tsv.Table {
	return map[plan.TableName]*tsv.Table{
		"timeline": {
			Header: []tsv.ColumnName{"西暦", "総収入", "総支出", "収支"},
			Rows: [][]string{
				{"2030", receipts[0], spending[0], balance[0]},
				{"2031", receipts[1], spending[1], balance[1]},
			},
		},
		"assets": {
			Header: []tsv.ColumnName{"西暦", "資産合計", "手が届く額", "不足"},
			Rows: [][]string{
				{"2030", total[0], total[0], shortfall[0]},
				{"2031", total[1], total[1], shortfall[1]},
			},
		},
		"tax": {
			Header: []tsv.ColumnName{"西暦", "合計"},
			Rows:   [][]string{{"2030", tax[0]}, {"2031", tax[1]}},
		},
	}
}

func assertTable(t *testing.T, got, want *tsv.Table) {
	t.Helper()

	if len(got.Header) != len(want.Header) {
		t.Fatalf("header = %v, want %v", got.Header, want.Header)
	}
	for i, name := range want.Header {
		if got.Header[i] != name {
			t.Fatalf("header = %v, want %v", got.Header, want.Header)
		}
	}
	if len(got.Rows) != len(want.Rows) {
		t.Fatalf("%d row(s), want %d: %v", len(got.Rows), len(want.Rows), got.Rows)
	}
	for i, row := range want.Rows {
		if len(got.Rows[i]) != len(row) {
			t.Fatalf("row %d = %v, want %v", i, got.Rows[i], row)
		}
		for j, field := range row {
			if got.Rows[i][j] != field {
				t.Errorf("row %d = %v, want %v", i, got.Rows[i], row)
				break
			}
		}
	}
}

func assertRow(t *testing.T, got *tsv.Table, at int, want []string) {
	t.Helper()

	if at >= len(got.Rows) {
		t.Fatalf("%d row(s), want at least %d", len(got.Rows), at+1)
	}
	if len(got.Rows[at]) != len(want) {
		t.Fatalf("row %d = %v, want %v", at, got.Rows[at], want)
	}
	for i, field := range want {
		if got.Rows[at][i] != field {
			t.Errorf("row %d = %v, want %v", at, got.Rows[at], want)
			return
		}
	}
}
