package actuals

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const PensionDir = "actuals/pension"

const (
	StandardRemunerationFile = "standard-remuneration.tsv"
	StandardBonusFile        = "standard-bonus.tsv"
)

const (
	RemunerationPersonColumn tsv.ColumnName = "本人"
	RemunerationYearColumn   tsv.ColumnName = "西暦"
	RemunerationMonthColumn  tsv.ColumnName = "月"
	RemunerationEraColumn    tsv.ColumnName = "和暦年月"
	RemunerationRecordColumn tsv.ColumnName = "記録"

	StandardRemunerationColumn tsv.ColumnName = "標準報酬月額[円]"
	StandardBonusColumn        tsv.ColumnName = "標準賞与額[円]"
)

type Remuneration struct {
	Person string
	Year   date.Year
	Month  int

	Amount money.Yen
	Known  bool
	Record string

	Bonus bool
}

func RemunerationRecord(root fs.FS) ([]Remuneration, error) {
	var all []Remuneration
	for _, f := range []struct {
		file   string
		column tsv.ColumnName
		bonus  bool
	}{
		{StandardRemunerationFile, StandardRemunerationColumn, false},
		{StandardBonusFile, StandardBonusColumn, true},
	} {
		body, err := fs.ReadFile(root, path.Join(PensionDir, f.file))
		if err != nil {
			return nil, fmt.Errorf("actuals.RemunerationRecord: %w", err)
		}
		table, err := tsv.Read(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("actuals.RemunerationRecord: %s: %w", f.file, err)
		}
		r, err := tsv.NewReader(table, tsv.Slot(path.Join(PensionDir, f.file)),
			RemunerationPersonColumn, RemunerationYearColumn, RemunerationMonthColumn,
			RemunerationEraColumn, f.column, RemunerationRecordColumn)
		if err != nil {
			return nil, err
		}

		for row := range r.Rows() {
			one := Remuneration{
				Person: r.Field(row, RemunerationPersonColumn),
				Record: r.Field(row, RemunerationRecordColumn),
				Bonus:  f.bonus,
			}
			if one.Year, err = r.Year(row, RemunerationYearColumn); err != nil {
				return nil, err
			}
			if one.Month, err = r.Count(row, RemunerationMonthColumn); err != nil {
				return nil, err
			}
			if one.Month < 1 || one.Month > date.MonthsAYear {
				return nil, r.Errorf(row, RemunerationMonthColumn, "月が 1〜12 でない: %d", one.Month)
			}

			if written := r.Field(row, f.column); written != "不明" {
				if one.Amount, err = money.ParseYen(written); err != nil {
					return nil, r.Errorf(row, f.column, "%v", err)
				}
				one.Known = true
			}
			all = append(all, one)
		}
	}
	return all, nil
}

func LatestRecorded(record []Remuneration, person string) (year date.Year, month int) {
	for _, one := range record {
		if one.Person != person {
			continue
		}
		if one.Year > year || (one.Year == year && one.Month > month) {
			year, month = one.Year, one.Month
		}
	}
	return year, month
}

const PensionRecordFile = "record.tsv"

const (
	PensionRecordPersonColumn tsv.ColumnName = "本人"
	PensionRecordAsOfColumn   tsv.ColumnName = "基準日"
	PensionRecordItemColumn   tsv.ColumnName = "項目"
	PensionRecordValueColumn  tsv.ColumnName = "値"
	PensionRecordUnitColumn   tsv.ColumnName = "単位"
)

const PaidNationalPensionMonthsItem = "国民年金 納付済月数"

const EmployeePensionMonthsItem = "厚生年金保険(一般) 加入月数"

type PensionRecordEntry struct {
	Person string
	AsOf   date.Date
	Item   string
	Value  int
	Unit   string
}

func PensionRecordOf(root fs.FS) ([]PensionRecordEntry, error) {
	body, err := fs.ReadFile(root, path.Join(PensionDir, PensionRecordFile))
	if err != nil {
		return nil, fmt.Errorf("actuals.PensionRecordOf: %w", err)
	}
	table, err := tsv.Read(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("actuals.PensionRecordOf: %s: %w", PensionRecordFile, err)
	}
	r, err := tsv.NewReader(table, tsv.Slot(path.Join(PensionDir, PensionRecordFile)),
		PensionRecordPersonColumn, PensionRecordAsOfColumn, PensionRecordItemColumn,
		PensionRecordValueColumn, PensionRecordUnitColumn)
	if err != nil {
		return nil, err
	}

	var all []PensionRecordEntry
	for row := range r.Rows() {
		one := PensionRecordEntry{
			Person: r.Field(row, PensionRecordPersonColumn),
			Item:   r.Field(row, PensionRecordItemColumn),
			Unit:   r.Field(row, PensionRecordUnitColumn),
		}
		if one.AsOf, err = date.Parse(r.Field(row, PensionRecordAsOfColumn)); err != nil {
			return nil, r.Errorf(row, PensionRecordAsOfColumn, "%v", err)
		}
		if one.Value, err = r.Count(row, PensionRecordValueColumn); err != nil {
			return nil, err
		}
		all = append(all, one)
	}
	return all, nil
}

func LatestPensionRecordItem(record []PensionRecordEntry, person, item string) (PensionRecordEntry, error) {
	var found PensionRecordEntry
	var ok bool
	for _, one := range record {
		if one.Person != person || one.Item != item {
			continue
		}
		if !ok || found.AsOf.Before(one.AsOf) {
			found, ok = one, true
		}
	}
	if !ok {
		return found, fmt.Errorf(
			"actuals.LatestPensionRecordItem: %s/%s に %q の %q が無い",
			PensionDir, PensionRecordFile, person, item)
	}
	return found, nil
}
