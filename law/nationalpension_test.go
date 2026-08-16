package law

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
	"github.com/Kuniwak/lifeplan/yeartest"
)

func nationalPensionTable(t *testing.T) NationalPensionPremiumTable {
	t.Helper()

	parsed := MustLoadNationalPensionPremiums(t, os.DirFS("../"+LawDirectory))
	return NationalPensionPremiumTable{YearYenTable: parsed}
}

func TestNationalPensionPremiumMonthly(t *testing.T) {
	type testCase struct {
		FiscalYear date.Year
		Expected   money.Yen
	}

	testCases := map[string]testCase{
		"the first year on record (boundary)":     {FiscalYear: 2005, Expected: 13_580},
		"a year the premium fell":                 {FiscalYear: 2012, Expected: 14_980},
		"the last year on record (boundary)":      {FiscalYear: 2023, Expected: 16_520},
		"past the record, the last figure stands": {FiscalYear: 2094, Expected: 18_290},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := nationalPensionTable(t).Monthly(tc.FiscalYear)

			if got != tc.Expected {
				t.Errorf("国民年金保険料: %d 年度は %d のはずだが %d になった", tc.FiscalYear, tc.Expected, got)
			}
		})
	}
}

func TestNationalPensionPremiumShouldMatchTheSpreadsheet(t *testing.T) {
	loaded := nationalPensionTable(t)
	table, err := sheets.New(os.DirFS("../testdata/sheets")).Table("national-pension-premium")
	if err != nil {
		t.Fatalf("Table: %v", err)
	}

	premiumColumn, ok := table.ColumnIndex("実際の保険料額[円]")
	if !ok {
		t.Fatal("no 実際の保険料額[円] column")
	}

	checked := 0
	yeartest.EachSheetsYear(t, table, func(year date.Year, row []string) {
		want, err := money.ParseYen(row[premiumColumn])
		if err != nil {
			t.Fatalf("%d: 実際の保険料額[円]: %v", year, err)
		}

		if got := loaded.Monthly(year); got != want {
			t.Errorf("%d年度: 国民年金保険料 = %d, want %d", year, got, want)
		}
		checked++
	})

	if checked == 0 {
		t.Fatal("no row was checked; the spreadsheet copy must have changed")
	}
	t.Logf("checked %d fiscal years", checked)
}
