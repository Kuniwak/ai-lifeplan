package actuals

import (
	"fmt"
	"path/filepath"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	TaxReturnPath          tsv.Slot = "actuals/tax-return/income-tax.tsv"
	TaxReturnHouseholdPath tsv.Slot = "actuals/tax-return/household.tsv"

	TaxReturnYearColumn   tsv.ColumnName = "西暦"
	TaxReturnItemColumn   tsv.ColumnName = "項目"
	TaxReturnAmountColumn tsv.ColumnName = "金額[円]"

	taxReturnYoungColumn    tsv.ColumnName = "23歳未満の扶養親族"
	taxReturnDisabledColumn tsv.ColumnName = "障害者である同一生計配偶者"

	ResidentTaxPath tsv.Slot = "actuals/resident-tax/resident-tax.tsv"

	ResidentTaxIncomeYearColumn        tsv.ColumnName = "所得年"
	ResidentTaxMunicipalIncomeColumn   tsv.ColumnName = "市所得割額[円]"
	ResidentTaxMunicipalPollColumn     tsv.ColumnName = "市均等割額[円]"
	ResidentTaxPrefecturalIncomeColumn tsv.ColumnName = "県所得割額[円]"
	ResidentTaxPrefecturalPollColumn   tsv.ColumnName = "県均等割額[円]"
)

type TaxReturn map[string]money.Yen

type TaxReturnHousehold struct {
	YoungDependant bool
	DisabledSpouse bool
}

func ReadTaxReturnHouseholdsUnder(root string) (map[date.Year]TaxReturnHousehold, error) {
	read, err := tsv.ReadFile(filepath.Join(root, string(TaxReturnHouseholdPath)))
	if err != nil {
		return nil, err
	}
	r, err := tsv.NewReader(read, TaxReturnHouseholdPath,
		TaxReturnYearColumn, taxReturnYoungColumn, taxReturnDisabledColumn)
	if err != nil {
		return nil, err
	}

	byYear := make(map[date.Year]TaxReturnHousehold, 8)
	for row := range r.Rows() {
		year, err := date.ParseYear(r.Field(row, TaxReturnYearColumn))
		if err != nil {
			return nil, r.Errorf(row, TaxReturnYearColumn, "%v", err)
		}
		if _, twice := byYear[year]; twice {
			return nil, r.Errorf(row, TaxReturnYearColumn, "%d が二度書かれている", year)
		}
		young, err := r.Bool(row, taxReturnYoungColumn)
		if err != nil {
			return nil, err
		}
		disabled, err := r.Bool(row, taxReturnDisabledColumn)
		if err != nil {
			return nil, err
		}
		byYear[year] = TaxReturnHousehold{YoungDependant: young, DisabledSpouse: disabled}
	}
	return byYear, nil
}

func (r TaxReturn) Has(items ...string) bool {
	for _, item := range items {
		if _, ok := r[item]; !ok {
			return false
		}
	}
	return true
}

func ReadTaxReturnsUnder(root string) (map[date.Year]TaxReturn, error) {
	read, err := tsv.ReadFile(filepath.Join(root, string(TaxReturnPath)))
	if err != nil {
		return nil, err
	}
	return TaxReturnItemsByYear(read)
}

func TaxReturnItemsByYear(taxReturn *tsv.Table) (map[date.Year]TaxReturn, error) {
	r, err := tsv.NewReader(taxReturn, TaxReturnPath,
		TaxReturnYearColumn, TaxReturnItemColumn, TaxReturnAmountColumn)
	if err != nil {
		return nil, err
	}

	byYear := make(map[date.Year]TaxReturn, 8)
	for row := range r.Rows() {
		year, err := date.ParseYear(r.Field(row, TaxReturnYearColumn))
		if err != nil {
			return nil, r.Errorf(row, TaxReturnYearColumn, "%v", err)
		}
		amount, err := money.ParseYen(r.Field(row, TaxReturnAmountColumn))
		if err != nil {
			return nil, r.Errorf(row, TaxReturnAmountColumn, "%v", err)
		}
		if byYear[year] == nil {
			byYear[year] = make(TaxReturn, 16)
		}
		item := r.Field(row, TaxReturnItemColumn)
		if _, twice := byYear[year][item]; twice {
			return nil, r.Errorf(row, TaxReturnItemColumn, "%d 年の %q が二度書かれている", year, item)
		}
		byYear[year][item] = amount
	}
	return byYear, nil
}

func residentTaxByIncomeYear(residentTax *tsv.Table) (map[date.Year]money.Yen, error) {
	records, err := ResidentTaxRecordsByIncomeYear(residentTax)
	if err != nil {
		return nil, err
	}

	byYear := make(map[date.Year]money.Yen, len(records))
	for year, record := range records {
		byYear[year] = record.Charged()
	}
	return byYear, nil
}

func incomeTaxFromItems(byItem map[string]money.Yen) (money.Yen, bool) {
	for _, item := range []string{
		"所得税及び復興特別所得税の額",
		"年調年税額",
		"源泉徴収税額",
	} {
		if amount, ok := byItem[item]; ok {
			return amount, true
		}
	}
	return 0, false
}

func ActualTakeHome(taxReturn, residentTax *tsv.Table) (relation.Table[money.Yen], error) {
	var empty relation.Table[money.Yen]

	items, err := TaxReturnItemsByYear(taxReturn)
	if err != nil {
		return empty, fmt.Errorf("actuals.ActualTakeHome: %w", err)
	}
	residents, err := residentTaxByIncomeYear(residentTax)
	if err != nil {
		return empty, fmt.Errorf("actuals.ActualTakeHome: %w", err)
	}

	rows := make([]relation.Row[money.Yen], 0, len(residents))
	for year, byItem := range items {
		salary, ok := byItem["給与収入"]
		if !ok {
			continue
		}
		social, ok := byItem["社会保険料控除"]
		if !ok {
			continue
		}
		incomeTax, ok := incomeTaxFromItems(byItem)
		if !ok {
			continue
		}
		resident, ok := residents[year]
		if !ok {
			continue
		}

		income := salary + byItem["業務雑所得の収入"]

		rows = append(rows, relation.Row[money.Yen]{
			Year:  year,
			Value: income - social - incomeTax - resident,
		})
	}

	return relation.New(rows), nil
}
