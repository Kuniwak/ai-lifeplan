package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type TaxTotalRow struct {
	IncomeTax   money.Yen
	IncomeTaxOf []PersonPremium

	ResidentTax   money.Yen
	ResidentTaxOf []PersonPremium

	PropertyTax, CityPlanningTax money.Yen

	Total money.Yen
}

type TaxTotalInput struct {
	IncomeTax   map[PersonName]relation.Table[IncomeTaxRow]
	ResidentTax map[PersonName]relation.Table[ResidentTaxRow]
	Property    relation.Table[PropertyTaxRow]
}

func TaxTotalTable(in TaxTotalInput) (relation.Table[TaxTotalRow], error) {
	var empty relation.Table[TaxTotalRow]

	years := in.Property.Years()
	rows := make([]relation.Row[TaxTotalRow], 0, len(years))

	for _, y := range years {
		var row TaxTotalRow

		for person, table := range in.IncomeTax {
			paid, ok := table.At(y)
			if !ok {
				return empty, fmt.Errorf("table.TaxTotalTable: %q has no income tax for %d", person, y)
			}
			row.IncomeTaxOf = append(row.IncomeTaxOf, PersonPremium{Name: person, Premium: paid.Payable})
			row.IncomeTax += paid.Payable
		}
		for person, table := range in.ResidentTax {
			paid, ok := table.At(y)
			if !ok {
				return empty, fmt.Errorf("table.TaxTotalTable: %q has no resident tax for %d", person, y)
			}
			row.ResidentTaxOf = append(row.ResidentTaxOf, PersonPremium{Name: person, Premium: paid.Total})
			row.ResidentTax += paid.Total
		}

		property, _ := in.Property.At(y)
		row.PropertyTax = property.PropertyTax
		row.CityPlanningTax = property.CityPlanningTax

		row.Total = row.IncomeTax + row.ResidentTax + row.PropertyTax + row.CityPlanningTax
		rows = append(rows, relation.Row[TaxTotalRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}
