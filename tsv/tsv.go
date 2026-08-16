package tsv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/wording"
)

const Delimiter = '\t'

type Table struct {
	Header []ColumnName
	Rows   [][]string
}

type Slot string

type ColumnName string

func (t *Table) ColumnIndex(name ColumnName) (int, bool) {
	for i, h := range t.Header {
		if h == name {
			return i, true
		}
	}
	return 0, false
}

func (t *Table) RowsByYear(context string, yearColumn ColumnName) (map[date.Year][]string, error) {
	at, ok := t.ColumnIndex(yearColumn)
	if !ok {
		return nil, fmt.Errorf("tsv: %s: no %q column", context, yearColumn)
	}

	byYear := make(map[date.Year][]string, len(t.Rows))
	for row, fields := range t.Rows {
		year, err := date.ParseYear(fields[at])
		if err != nil {
			return nil, fmt.Errorf("tsv: %s: row %d, column %q: %w", context, row+1, yearColumn, err)
		}
		if _, twice := byYear[year]; twice {
			return nil, wording.DuplicateKeyError(fmt.Sprintf("tsv: %s", context),
				string(yearColumn), wording.Number(year),
				wording.WhichRowReadsTheYearEn+" (one row per year, keyed by 西暦)")
		}
		byYear[year] = fields
	}
	return byYear, nil
}

func Read(r io.Reader) (*Table, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("tsv.Read: %w", err)
	}
	if err := AssertUTF8(content); err != nil {
		return nil, fmt.Errorf("tsv.Read: %w", err)
	}

	cr := csv.NewReader(bytes.NewReader(StripByteOrderMark(content)))
	cr.Comma = Delimiter
	cr.FieldsPerRecord = 0

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("tsv.Read: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("tsv.Read: no header row (the columns of an empty file are unknown)")
	}

	header := make([]ColumnName, 0, len(records[0]))
	for _, name := range records[0] {
		header = append(header, ColumnName(name))
	}
	if err := validateHeader(header); err != nil {
		return nil, fmt.Errorf("tsv.Read: %w", err)
	}

	table := &Table{Header: header}
	if len(records) > 1 {
		table.Rows = records[1:]
	}
	return table, nil
}

func Write(w io.Writer, t *Table) error {
	if err := validateHeader(t.Header); err != nil {
		return fmt.Errorf("tsv.Write: %w", err)
	}

	for i, row := range t.Rows {
		if len(row) != len(t.Header) {
			return fmt.Errorf("tsv.Write: row %d has %d fields, but the header has %d", i+1, len(row), len(t.Header))
		}
	}

	cw := csv.NewWriter(w)
	cw.Comma = Delimiter

	header := make([]string, 0, len(t.Header))
	for _, name := range t.Header {
		header = append(header, string(name))
	}
	if err := writeRow(w, cw, header); err != nil {
		return fmt.Errorf("tsv.Write: header: %w", err)
	}
	for i, row := range t.Rows {
		if err := writeRow(w, cw, row); err != nil {
			return fmt.Errorf("tsv.Write: row %d: %w", i+1, err)
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("tsv.Write: %w", err)
	}
	return nil
}

func writeRow(w io.Writer, cw *csv.Writer, row []string) error {
	if len(row) == 1 && row[0] == "" {
		cw.Flush()
		if err := cw.Error(); err != nil {
			return err
		}
		_, err := io.WriteString(w, "\"\"\n")
		return err
	}
	return cw.Write(row)
}

func validateHeader(header []ColumnName) error {
	if len(header) == 0 {
		return fmt.Errorf("no header row")
	}

	seen := make(map[ColumnName]struct{}, len(header))
	for i, name := range header {
		if name == "" {
			return fmt.Errorf("column %d has an empty name", i+1)
		}
		if _, dup := seen[name]; dup {
			return fmt.Errorf("column name %q appears more than once, so a reference to it is ambiguous", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

var byteOrderMark = []byte{0xEF, 0xBB, 0xBF}

func StripByteOrderMark(content []byte) []byte {
	return bytes.TrimPrefix(content, byteOrderMark)
}

func AssertUTF8(content []byte) error {
	if utf8.Valid(content) {
		return nil
	}
	return fmt.Errorf(
		"the file is not UTF-8; convert it first with: iconv -f CP932 -t UTF-8 <file> > <file>.utf8")
}
