package tsv

import (
	"fmt"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"slices"
	"strconv"
	"strings"
)

type Reader struct {
	context Slot
	table   *Table
	index   map[ColumnName]int
}

func NewReader(table *Table, context Slot, columns ...ColumnName) (Reader, error) {
	if table == nil {
		return Reader{}, fmt.Errorf("tsv: %s: the table is not there", context)
	}

	index := make(map[ColumnName]int, len(columns))
	for _, column := range columns {
		i, ok := table.ColumnIndex(column)
		if !ok {
			return Reader{}, fmt.Errorf("tsv: %s: no %q column", context, column)
		}
		index[column] = i
	}
	return Reader{context: context, table: table, index: index}, nil
}

func (r Reader) Slot() Slot { return r.context }

func (r Reader) Rows() int { return len(r.table.Rows) }

func (r Reader) Field(row int, column ColumnName) string {
	i, asked := r.index[column]
	if !asked {
		panic(fmt.Sprintf("tsv: %s: %q was not one of the columns this reader was opened with (%s)",
			r.context, column, strings.Join(r.columnNames(), ", ")))
	}
	return r.table.Rows[row][i]
}

func (r Reader) columnNames() []string {
	names := make([]string, 0, len(r.index))
	for column := range r.index {
		names = append(names, string(column))
	}
	slices.Sort(names)
	return names
}

func (r Reader) Errorf(row int, column ColumnName, format string, args ...any) error {
	return fmt.Errorf("tsv: %s: row %d, column %q: %s", r.context, row+1, column, fmt.Sprintf(format, args...))
}

func (r Reader) Count(row int, column ColumnName) (int, error) {
	field := r.Field(row, column)
	n, err := strconv.Atoi(strings.TrimSpace(field))
	if err != nil {
		return 0, r.Errorf(row, column, "%q is not a whole number", field)
	}
	return n, nil
}

func (r Reader) Months(row int, column ColumnName) (date.Months, error) {
	months, err := date.ParseMonths(r.Field(row, column))
	if err != nil {
		return date.NoMonths, r.Errorf(row, column, "%v", err)
	}
	return months, nil
}

func (r Reader) Bool(row int, column ColumnName) (bool, error) {
	switch field := strings.TrimSpace(r.Field(row, column)); field {
	case "TRUE":
		return true, nil
	case "FALSE":
		return false, nil
	default:
		return false, r.Errorf(row, column, "%q is neither TRUE nor FALSE", field)
	}
}

func (r Reader) Yen(row int, column ColumnName) (money.Yen, error) {
	amount, err := money.ParseYen(r.Field(row, column))
	if err != nil {
		return 0, r.Errorf(row, column, "%v", err)
	}
	return amount, nil
}

func (r Reader) Percent(row int, column ColumnName) (money.Rate, error) {
	rate, err := money.ParsePercent(r.Field(row, column))
	if err != nil {
		return money.Rate{}, r.Errorf(row, column, "%v", err)
	}
	return rate, nil
}

func (r Reader) Year(row int, column ColumnName) (date.Year, error) {
	y, err := date.ParseYear(r.Field(row, column))
	if err != nil {
		return 0, r.Errorf(row, column, "%v", err)
	}
	return y, nil
}

func (r Reader) PriceMove(row int, column ColumnName) (money.PriceMove, error) {
	move, err := money.ParsePriceMove(r.Field(row, column))
	if err != nil {
		return money.PriceMove{}, r.Errorf(row, column, "%v", err)
	}
	return move, nil
}
