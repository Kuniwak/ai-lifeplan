package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func taxTotalOfTheBaseProject(t *testing.T) relation.Table[table.TaxTotalRow] {
	t.Helper()

	built, err := table.TaxTotalTable(table.TaxTotalInput{
		IncomeTax: map[table.PersonName]relation.Table[table.IncomeTaxRow]{
			"夫": incomeTaxOfTheBaseProject(t, "夫", "妻"),
			"妻": incomeTaxOfTheBaseProject(t, "妻", ""),
		},
		ResidentTax: map[table.PersonName]relation.Table[table.ResidentTaxRow]{
			"夫": residentTaxOfTheBaseProject(t, "夫", "妻"),
			"妻": residentTaxOfTheBaseProject(t, "妻", ""),
		},
		Property: propertyTaxOfTheBaseProject(t),
	})
	if err != nil {
		t.Fatalf("table.TaxTotalTable: %v", err)
	}
	return built
}
