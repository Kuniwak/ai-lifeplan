package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
)

func residentTaxMunicipalRecords(t *testing.T) map[date.Year]actuals.ResidentTaxRecord {
	t.Helper()

	read, err := tsv.ReadFile("../actuals/resident-tax/resident-tax.tsv")
	if err != nil {
		t.Fatalf("tsv.ReadFile: %v", err)
	}
	records, err := actuals.ResidentTaxRecordsByIncomeYear(read)
	if err != nil {
		t.Fatalf("actuals.ResidentTaxRecordsByIncomeYear: %v", err)
	}
	return records
}
