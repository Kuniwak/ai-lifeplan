package sheets

import (
	"fmt"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const VerificationTable = "verification"

const manYenPerMonth = "万/月"

type VerificationRow struct {
	Name   string
	Got    money.Yen
	Want   money.Yen
	Within money.Yen
}

func (r VerificationRow) Failed() bool {
	diff := r.Got - r.Want
	return diff > r.Within || diff < -r.Within
}

func (c Copy) Verification() ([]VerificationRow, error) {
	table, err := c.Table(VerificationTable)
	if err != nil {
		return nil, err
	}

	index := func(name tsv.ColumnName) (int, error) {
		at, ok := table.ColumnIndex(name)
		if !ok {
			return 0, fmt.Errorf("sheets.Verification: no %s column", name)
		}
		return at, nil
	}
	nameAt, err := index("名目")
	if err != nil {
		return nil, err
	}
	gotAt, err := index("計算結果")
	if err != nil {
		return nil, err
	}
	wantAt, err := index("期待結果")
	if err != nil {
		return nil, err
	}
	withinAt, err := index("許容誤差")
	if err != nil {
		return nil, err
	}

	rows := make([]VerificationRow, 0, len(table.Rows))
	for _, fields := range table.Rows {
		row := VerificationRow{Name: fields[nameAt]}
		for _, c := range []struct {
			column tsv.ColumnName
			at     int
			into   *money.Yen
		}{
			{"計算結果", gotAt, &row.Got},
			{"期待結果", wantAt, &row.Want},
			{"許容誤差", withinAt, &row.Within},
		} {
			v, err := ManYen(strings.TrimSuffix(fields[c.at], manYenPerMonth))
			if err != nil {
				return nil, fmt.Errorf("sheets.Verification: %s の %s: %w", row.Name, c.column, err)
			}
			*c.into = v
		}
		rows = append(rows, row)
	}
	return rows, nil
}
