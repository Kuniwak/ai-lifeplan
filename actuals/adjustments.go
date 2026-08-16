package actuals

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const AdjustmentsPath tsv.Slot = "actuals/adjustments.tsv"

const (
	AdjustmentMonthColumn  = CashflowMonthColumn
	AdjustmentItemColumn   = CashflowItemColumn
	AdjustmentAmountColumn = CashflowAmountColumn

	AdjustmentSourceColumn tsv.ColumnName = "出典"
)

func ReadAdjustments(table *tsv.Table) (*tsv.Table, error) {
	at := make(map[tsv.ColumnName]int, 4)
	for _, column := range []tsv.ColumnName{
		AdjustmentMonthColumn, AdjustmentItemColumn, AdjustmentAmountColumn, AdjustmentSourceColumn,
	} {
		i, ok := table.ColumnIndex(column)
		if !ok {
			return nil, fmt.Errorf("actuals.ReadAdjustments: no %q column", column)
		}
		at[column] = i
	}

	out := &tsv.Table{Header: []tsv.ColumnName{CashflowMonthColumn, CashflowItemColumn, CashflowAmountColumn}}
	for row, fields := range table.Rows {
		month := fields[at[AdjustmentMonthColumn]]
		if !isMonth(month) {
			return nil, fmt.Errorf(
				"actuals.ReadAdjustments: row %d: 年月が %q である。yyyy-mm と書くこと", row+1, month)
		}

		item := fields[at[AdjustmentItemColumn]]
		if item == "" {
			return nil, fmt.Errorf("actuals.ReadAdjustments: row %d: 費目が空である", row+1)
		}

		amount, err := money.ParseYen(fields[at[AdjustmentAmountColumn]])
		if err != nil {
			return nil, fmt.Errorf("actuals.ReadAdjustments: row %d: %w", row+1, err)
		}
		if amount == 0 {
			return nil, fmt.Errorf(
				"actuals.ReadAdjustments: row %d: 金額が 0 である。足すもののない行を書く理由が無い", row+1)
		}

		if fields[at[AdjustmentSourceColumn]] == "" {
			return nil, fmt.Errorf(
				"actuals.ReadAdjustments: row %d: %s が空である。手で足す行には要る",
				row+1, AdjustmentSourceColumn)
		}

		out.Rows = append(out.Rows, []string{month, item, fmt.Sprint(int64(amount))})
	}
	return out, nil
}

const (
	SourceFileColumn  tsv.ColumnName = "ファイル"
	SourceBytesColumn tsv.ColumnName = "バイト数"
	SourceHashColumn  tsv.ColumnName = "sha256"
	SourceRowsColumn  tsv.ColumnName = "レコード数"
)

const SourcesPath tsv.Slot = "actuals/sources.tsv"

type Source struct {
	Name  string
	Bytes int
	Hash  string
	Rows  int
}

func SourceTable(sources []Source) *tsv.Table {
	out := &tsv.Table{Header: []tsv.ColumnName{
		SourceFileColumn, SourceBytesColumn, SourceHashColumn, SourceRowsColumn,
	}}
	for _, s := range sources {
		out.Rows = append(out.Rows, []string{
			s.Name, fmt.Sprint(s.Bytes), s.Hash, fmt.Sprint(s.Rows),
		})
	}
	return out
}

const ExcludedObservationColumn tsv.ColumnName = "観測"

func CountExcluded(readers []io.Reader, categories ImportRules, table *tsv.Table) (*tsv.Table, error) {
	var records []exportRecord
	for _, r := range readers {
		read, err := readExportRecords(r)
		if err != nil {
			return nil, fmt.Errorf("actuals.CountExcluded: %w", err)
		}
		records = append(records, read...)
	}

	at, ok := table.ColumnIndex(ExcludedObservationColumn)
	if !ok {
		return nil, fmt.Errorf("actuals.CountExcluded: no %q column", ExcludedObservationColumn)
	}
	if len(table.Rows) != len(categories.byExcluded) {
		return nil, fmt.Errorf(
			"actuals.CountExcluded: 表は %d 行だが規則は %d 個である。同じ表から作られていない",
			len(table.Rows), len(categories.byExcluded))
	}

	type tally struct {
		rows, marked int
		total        money.Yen
	}
	counted := make(map[Rule]*tally, len(categories.byExcluded))

	for _, record := range records {
		if record.counted || record.amount == 0 {
			continue
		}
		if categories.held.IsMoveInto(record.account, record.content, record.month, DiscardUsed) {
			continue
		}
		if _, untrusted := categories.held.BalanceUntrusted(record.account); untrusted {
			continue
		}
		for _, e := range categories.byExcluded {
			if !e.matches(record.major, record.minor, record.account, record.month, record.marked) {
				continue
			}
			t := counted[e.rule]
			if t == nil {
				t = &tally{}
				counted[e.rule] = t
			}
			t.rows++
			t.total += record.amount
			if record.marked {
				t.marked++
			}
			break
		}
	}

	out := &tsv.Table{Header: slices.Clone(table.Header), Rows: make([][]string, 0, len(table.Rows))}
	for i, fields := range table.Rows {
		row := slices.Clone(fields)
		t := counted[categories.byExcluded[i].rule]
		if t == nil {
			t = &tally{}
		}
		row[at] = fmt.Sprintf(
			"全 %d 件・%s 円。うち MoneyForward の振替の印があるもの %d 件（この行は書き出しを数えて作った）",
			t.rows, withSeparators(int64(t.total)), t.marked)
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

const PayeeObservationColumn tsv.ColumnName = "観測"

func CountPayees(readers []io.Reader, categories ImportRules, table *tsv.Table) (*tsv.Table, error) {
	var records []exportRecord
	for _, r := range readers {
		read, err := readExportRecords(r)
		if err != nil {
			return nil, fmt.Errorf("actuals.CountPayees: %w", err)
		}
		records = append(records, read...)
	}

	at, ok := table.ColumnIndex(PayeeObservationColumn)
	if !ok {
		return nil, fmt.Errorf("actuals.CountPayees: no %q column", PayeeObservationColumn)
	}
	if len(table.Rows) != len(categories.byPayee) {
		return nil, fmt.Errorf(
			"actuals.CountPayees: 表は %d 行だが規則は %d 個である。同じ表から作られていない",
			len(table.Rows), len(categories.byPayee))
	}

	type tally struct {
		rows  int
		total money.Yen
	}
	counted := make(map[Rule]*tally, len(categories.byPayee))

	for _, record := range records {
		if !record.counted || record.marked || record.amount == 0 {
			continue
		}
		if categories.held.IsMoveInto(record.account, record.content, record.month, DiscardUsed) {
			continue
		}
		if _, untrusted := categories.held.BalanceUntrusted(record.account); untrusted {
			continue
		}
		for _, p := range categories.byPayee {
			if !p.matches(record.account, record.major, record.content, record.month) {
				continue
			}
			t := counted[p.rule]
			if t == nil {
				t = &tally{}
				counted[p.rule] = t
			}
			t.rows++
			t.total += record.amount
			break
		}
	}

	out := &tsv.Table{Header: slices.Clone(table.Header), Rows: make([][]string, 0, len(table.Rows))}
	for i, fields := range table.Rows {
		row := slices.Clone(fields)
		t := counted[categories.byPayee[i].rule]
		if t == nil {
			t = &tally{}
		}
		row[at] = fmt.Sprintf(
			"全 %d 件・%s 円（この行は書き出しを数えて作った）", t.rows, withSeparators(int64(t.total)))
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func withSeparators(n int64) string {
	s := fmt.Sprint(n)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}
