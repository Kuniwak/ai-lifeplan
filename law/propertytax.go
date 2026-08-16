package law

import (
	"fmt"
	"io/fs"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	PropertyTaxRateColumn      tsv.ColumnName = "固定資産税率"
	CityPlanningTaxRateColumn  tsv.ColumnName = "都市計画税率"
	ResidentialLandFixedColumn tsv.ColumnName = "住宅用地特例分母（固定資産税）"
	ResidentialLandCityColumn  tsv.ColumnName = "住宅用地特例分母（都市計画税）"
	NewHouseReliefRateColumn   tsv.ColumnName = "新築住宅減額割合"
	NewHouseReliefYearsColumn  tsv.ColumnName = "新築住宅減額年数"
)

const (
	PropertyTaxBaseUnit money.Yen = 1_000

	PropertyTaxUnit money.Yen = 100
)

type PropertyTaxTerms struct {
	PropertyRate, CityPlanningRate money.Rate

	ResidentialLandFixed, ResidentialLandCity int

	NewHouseReliefRate  money.Rate
	NewHouseReliefYears int
}

type PropertyTaxTable struct {
	terms YearTable[PropertyTaxTerms]
}

const PropertyTaxTableName = "property-tax"

func LoadPropertyTaxTable(fsys fs.FS, municipality Municipality) (PropertyTaxTable, error) {
	table, err := LoadRegionalTable(fsys, string(municipality), PropertyTaxTableName)
	if err != nil {
		return PropertyTaxTable{}, fmt.Errorf("law.LoadPropertyTaxTable: %w", err)
	}
	return ParsePropertyTaxTable(table)
}

func ParsePropertyTaxTable(table *tsv.Table) (PropertyTaxTable, error) {
	r, err := newReader(table, PropertyTaxTableName, PropertyTaxRateColumn, CityPlanningTaxRateColumn,
		ResidentialLandFixedColumn, ResidentialLandCityColumn,
		NewHouseReliefRateColumn, NewHouseReliefYearsColumn,
		LawStartYearColumn, LawEndYearColumn)
	if err != nil {
		return PropertyTaxTable{}, fmt.Errorf("law.ParsePropertyTaxTable: %w", err)
	}

	periods, err := readYearPeriods(r, func(row int) (PropertyTaxTerms, error) {
		var terms PropertyTaxTerms
		for _, read := range []struct {
			column tsv.ColumnName
			into   *money.Rate
		}{
			{PropertyTaxRateColumn, &terms.PropertyRate},
			{CityPlanningTaxRateColumn, &terms.CityPlanningRate},
			{NewHouseReliefRateColumn, &terms.NewHouseReliefRate},
		} {
			var err error
			if *read.into, err = r.Percent(row, read.column); err != nil {
				return terms, err
			}
		}
		for _, read := range []struct {
			column tsv.ColumnName
			into   *int
		}{
			{ResidentialLandFixedColumn, &terms.ResidentialLandFixed},
			{ResidentialLandCityColumn, &terms.ResidentialLandCity},
			{NewHouseReliefYearsColumn, &terms.NewHouseReliefYears},
		} {
			var err error
			if *read.into, err = r.Count(row, read.column); err != nil {
				return terms, err
			}
		}
		return terms, nil
	})
	if err != nil {
		return PropertyTaxTable{}, fmt.Errorf("law.ParsePropertyTaxTable: %w", err)
	}

	terms, err := NewYearTableOfPeriods(periods)
	if err != nil {
		return PropertyTaxTable{}, fmt.Errorf("law.ParsePropertyTaxTable: %w", err)
	}
	return PropertyTaxTable{terms: terms}, nil
}

type PropertyTaxBill struct {
	LandBaseProperty, LandBaseCityPlanning money.Yen

	HouseBase money.Yen

	PropertyTax, CityPlanningTax, NewHouseRelief money.Yen

	Total money.Yen
}

func (t PropertyTaxTable) Bill(landValue, houseBase money.Yen, yearsSinceBuilt int, year date.Year) (PropertyTaxBill, error) {
	terms, ok := t.terms.At(year)
	if !ok {
		return PropertyTaxBill{}, fmt.Errorf("law.PropertyTaxTable.Bill: nothing is written about %d", year)
	}

	bill := PropertyTaxBill{
		LandBaseProperty:     landValue / money.Yen(terms.ResidentialLandFixed),
		LandBaseCityPlanning: landValue / money.Yen(terms.ResidentialLandCity),
		HouseBase:            houseBase,
	}

	property := (bill.LandBaseProperty + bill.HouseBase).Truncate(PropertyTaxBaseUnit).
		Mul(terms.PropertyRate, money.Truncate)

	if yearsSinceBuilt >= 0 && yearsSinceBuilt < terms.NewHouseReliefYears {
		bill.NewHouseRelief = bill.HouseBase.Mul(terms.PropertyRate, money.Truncate).
			Mul(terms.NewHouseReliefRate, money.Truncate)
	}
	bill.PropertyTax = max(property-bill.NewHouseRelief, 0).Truncate(PropertyTaxUnit)

	bill.CityPlanningTax = (bill.LandBaseCityPlanning + bill.HouseBase).Truncate(PropertyTaxBaseUnit).
		Mul(terms.CityPlanningRate, money.Truncate).Truncate(PropertyTaxUnit)

	bill.Total = bill.PropertyTax + bill.CityPlanningTax
	return bill, nil
}
