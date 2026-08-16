package actuals

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const PayslipDir = "actuals/payslip"

const (
	PayslipPersonColumn tsv.ColumnName = "本人"

	PayslipYearColumn     tsv.ColumnName = "年"
	PayslipMonthColumn    tsv.ColumnName = "月"
	PayslipEmployerColumn tsv.ColumnName = "勤務先"
	PayslipKindColumn     tsv.ColumnName = "名目"

	PayslipGrossColumn      tsv.ColumnName = "総支給[円]"
	PayslipHealthColumn     tsv.ColumnName = "健康保険料[円]"
	PayslipPensionColumn    tsv.ColumnName = "厚生年金保険料[円]"
	PayslipEmploymentColumn tsv.ColumnName = "雇用保険料[円]"
	PayslipIncomeTaxColumn  tsv.ColumnName = "所得税[円]"
	PayslipResidentColumn   tsv.ColumnName = "住民税[円]"
)

var PayslipColumns = []tsv.ColumnName{
	PayslipPersonColumn,
	PayslipYearColumn, PayslipMonthColumn, PayslipEmployerColumn, PayslipKindColumn,
	PayslipGrossColumn, PayslipHealthColumn, PayslipPensionColumn,
	PayslipEmploymentColumn, PayslipIncomeTaxColumn, PayslipResidentColumn,
}

type PayslipKind string

const (
	PayslipSalary PayslipKind = "給与"

	PayslipBonus PayslipKind = "賞与"

	PayslipPreviousEmployer PayslipKind = "前職"
)

type Payslip struct {
	Person string

	Year     date.Year
	Month    int
	Employer string

	Kind PayslipKind

	Gross                                               money.Yen
	Health, Pension, Employment, IncomeTax, ResidentTax money.Yen
}

func Payslips(dir fs.FS) ([]Payslip, error) {
	names, err := fs.Glob(dir, "*.tsv")
	if err != nil {
		return nil, fmt.Errorf("actuals.Payslips: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("actuals.Payslips: no TSV in the payslip directory")
	}

	var slips []Payslip
	for _, name := range names {
		file, err := fs.ReadFile(dir, name)
		if err != nil {
			return nil, fmt.Errorf("actuals.Payslips: %w", err)
		}
		table, err := tsv.Read(bytes.NewReader(file))
		if err != nil {
			return nil, fmt.Errorf("actuals.Payslips: %s: %w", name, err)
		}

		slot := tsv.Slot(path.Join(PayslipDir, name))
		r, err := tsv.NewReader(table, slot, PayslipColumns...)
		if err != nil {
			return nil, err
		}

		for row := range r.Rows() {
			slip := Payslip{
				Person:   r.Field(row, PayslipPersonColumn),
				Employer: r.Field(row, PayslipEmployerColumn),
				Kind:     PayslipKind(r.Field(row, PayslipKindColumn)),
			}
			if slip.Year, err = r.Year(row, PayslipYearColumn); err != nil {
				return nil, err
			}
			if slip.Month, err = r.Count(row, PayslipMonthColumn); err != nil {
				return nil, err
			}
			if slip.Month < 1 || slip.Month > date.MonthsAYear {
				return nil, r.Errorf(row, PayslipMonthColumn, "月が 1〜12 でない: %d", slip.Month)
			}
			for _, field := range []struct {
				into   *money.Yen
				column tsv.ColumnName
			}{
				{&slip.Gross, PayslipGrossColumn},
				{&slip.Health, PayslipHealthColumn},
				{&slip.Pension, PayslipPensionColumn},
				{&slip.Employment, PayslipEmploymentColumn},
				{&slip.IncomeTax, PayslipIncomeTaxColumn},
				{&slip.ResidentTax, PayslipResidentColumn},
			} {
				amount, err := money.ParseYen(r.Field(row, field.column))
				if err != nil {
					return nil, r.Errorf(row, field.column, "%v", err)
				}
				*field.into = amount
			}
			slips = append(slips, slip)
		}
	}

	slices.SortFunc(slips, ComparePayslips)
	return slips, nil
}

func ComparePayslips(a, b Payslip) int {
	if c := cmp.Compare(a.Person, b.Person); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Year, b.Year); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Month, b.Month); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Employer, b.Employer); c != 0 {
		return c
	}
	return cmp.Compare(a.Kind, b.Kind)
}

func PayslipsUnder(root fs.FS) ([]Payslip, error) {
	dir, err := fs.Sub(root, PayslipDir)
	if err != nil {
		return nil, fmt.Errorf("actuals.PayslipsUnder: %w", err)
	}
	return Payslips(dir)
}

func ResidentTaxWithheldByLevyYear(slips []Payslip) map[date.Year]money.Yen {
	withheld := make(map[date.Year]money.Yen, 8)
	for _, slip := range slips {
		levy := date.Year(slip.Year)
		if slip.Month < 6 {
			levy--
		}
		withheld[levy] += slip.ResidentTax
	}
	return withheld
}
