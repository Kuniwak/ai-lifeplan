package law

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
	"github.com/Kuniwak/lifeplan/tsv"
)

func municipalitiesWithTables(t *testing.T) []Municipality {
	t.Helper()

	regions, err := RegionsWith(os.DirFS("../"+LawDirectory), ResidentRateTableName)
	if err != nil {
		t.Fatalf("RegionsWith: %v", err)
	}
	if len(regions) == 0 {
		t.Fatal("data/law has no municipality with resident tax figures")
	}

	municipalities := make([]Municipality, 0, len(regions))
	for _, region := range regions {
		municipalities = append(municipalities, Municipality(region))
	}
	return municipalities
}

func residentTablesAt(t *testing.T, municipality Municipality, year date.Year) (ResidentRates, ResidentPerCapitaLevy, ResidentExemption) {
	t.Helper()

	return MustLoadResidentLevies(t, os.DirFS("../"+LawDirectory), municipality).
		MustTablesFor(t, municipality).
		MustAt(t, year)
}

func setagaya(t *testing.T) ResidentRates {
	t.Helper()
	rates, _, _ := residentTablesAt(t, "世田谷区", 2023)
	return rates
}

func setagayaExemption(t *testing.T) ResidentExemption {
	t.Helper()
	_, _, exemption := residentTablesAt(t, "世田谷区", 2023)
	return exemption
}

func TestTheResidentTaxWorksOutIncomeTheSameWayAsTheIncomeTax(t *testing.T) {
	table, err := sheets.New(os.DirFS("../testdata/sheets")).Table("resident-tax-income")
	if err != nil {
		t.Fatalf("Table: %v", err)
	}
	salaryColumn, ok := table.ColumnIndex("収入金額")
	if !ok {
		t.Fatal("no 収入金額 column")
	}
	incomeColumn, ok := table.ColumnIndex("所得金額")
	if !ok {
		t.Fatal("no 所得金額 column")
	}

	checked := 0
	for i, row := range table.Rows {
		salary, err := parseManYen(row[salaryColumn])
		if err != nil {
			t.Fatalf("row %d: 収入金額: %v", i+1, err)
		}
		want, err := parseManYen(row[incomeColumn])
		if err != nil {
			t.Fatalf("row %d: 所得金額: %v", i+1, err)
		}

		if got := SalaryIncome(salary, 2024); got != want {
			t.Errorf("収入 %s万: SalaryIncome gives %d, the resident tax table says %d",
				row[salaryColumn], got, want)
		}
		checked++
	}

	if checked < 800 {
		t.Errorf("only %d rows were checked; the golden table looks truncated", checked)
	}
}

func TestTheResidentTaxPensionIncomeMatchesTheIncomeTax(t *testing.T) {
	type testCase struct {
		Block string
		Age   int
	}

	testCases := map[string]testCase{
		"under 65":    {Block: "resident-tax-pension-income-under65", Age: 64},
		"65 and over": {Block: "resident-tax-pension-income-over65", Age: 65},
	}

	tiers := []struct {
		Column      tsv.ColumnName
		TotalIncome money.Yen
	}{
		{Column: "合計所得1000万円以下", TotalIncome: 5_000_000},
		{Column: "合計所得2000万円以下", TotalIncome: 15_000_000},
		{Column: "合計所得2000超", TotalIncome: 25_000_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			table, err := sheets.New(os.DirFS("../testdata/sheets")).Table(tc.Block)
			if err != nil {
				t.Fatalf("Table: %v", err)
			}
			receivedColumn, ok := table.ColumnIndex("収入金額")
			if !ok {
				t.Fatal("no 収入金額 column")
			}

			for _, tier := range tiers {
				tierColumn, ok := table.ColumnIndex(tier.Column)
				if !ok {
					t.Fatalf("no %s column", tier.Column)
				}

				for i, row := range table.Rows {
					received, err := parseManYen(row[receivedColumn])
					if err != nil {
						t.Fatalf("row %d: 収入金額: %v", i+1, err)
					}
					want, err := parseManYen(row[tierColumn])
					if err != nil {
						t.Fatalf("row %d: %s: %v", i+1, tier.Column, err)
					}

					if got := PensionIncome(received, tc.Age, tier.TotalIncome, 2024); got != want {
						t.Errorf("年金収入 %s万, age %d, %s: income %d, the spreadsheet says %d",
							row[receivedColumn], tc.Age, tier.Column, got, want)
					}
				}
			}
		})
	}
}

func TestResidentTaxOf(t *testing.T) {
	type testCase struct {
		Taxable         money.Yen
		WantPerCapita   money.Yen
		WantPrefectural money.Yen
		WantMunicipal   money.Yen
	}

	testCases := map[string]testCase{
		"an ordinary taxable income": {
			Taxable: 5_000_000, WantPerCapita: 5_000, WantPrefectural: 200_000, WantMunicipal: 300_000,
		},
		"nothing to tax still owes the per capita levy (boundary value)": {
			Taxable: 0, WantPerCapita: 5_000, WantPrefectural: 0, WantMunicipal: 0,
		},
		"below a thousand yen is truncated away (boundary value)": {
			Taxable: 999, WantPerCapita: 5_000, WantPrefectural: 0, WantMunicipal: 0,
		},
		"the first thousand (boundary value)": {
			Taxable: 1_000, WantPerCapita: 5_000, WantPrefectural: 40, WantMunicipal: 60,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			m := setagaya(t)

			got := ResidentTaxOf(tc.Taxable, m)

			if _, levy, _ := residentTablesAt(t, "世田谷区", 2023); levy.Total() != tc.WantPerCapita {
				t.Errorf("均等割 = %d, want %d", levy.Total(), tc.WantPerCapita)
			}
			if got.Prefectural != tc.WantPrefectural {
				t.Errorf("Prefectural = %d, want %d", got.Prefectural, tc.WantPrefectural)
			}
			if got.Municipal != tc.WantMunicipal {
				t.Errorf("Municipal = %d, want %d", got.Municipal, tc.WantMunicipal)
			}
			if want := got.Prefectural + got.Municipal; got.Total() != want {
				t.Errorf("Total() = %d, want %d", got.Total(), want)
			}
		})
	}
}

func TestLoadResidentRateTables(t *testing.T) {
	for _, name := range municipalitiesWithTables(t) {
		if _, err := LoadResidentRateTables(os.DirFS("../"+LawDirectory), name); err != nil {
			t.Errorf("LoadResidentRateTables(%q): %v", name, err)
		}
	}

	_, err := LoadResidentRateTables(os.DirFS("../"+LawDirectory), "札幌市")
	if err == nil {
		t.Fatal("want error for a municipality with no figures, got none")
	}
	if got := err.Error(); got == "" {
		t.Error("the error says nothing")
	}
}

func TestEveryMunicipalityShouldHavePlausibleFigures(t *testing.T) {
	for _, name := range municipalitiesWithTables(t) {
		m, levy, _ := residentTablesAt(t, name, 2023)

		if levy.Prefectural <= 0 || levy.Municipal <= 0 {
			t.Errorf("%s has a non-positive per capita levy", name)
		}
		if got := ResidentTaxOf(10_000_000, m); got.Prefectural <= 0 || got.Municipal <= 0 {
			t.Errorf("%s levies nothing on an income of ten million: %+v", name, got)
		}
	}
}

func TestResidentTaxShouldRiseWithTheTaxableIncome(t *testing.T) {
	tables := loadedMunicipalities(t)

	rapid.Check(t, func(t *rapid.T) {
		m := tables[rapid.IntRange(0, len(tables)-1).Draw(t, "municipality")]
		a := money.Yen(rapid.Int64Range(0, 200_000).Draw(t, "a")) * 1_000
		b := money.Yen(rapid.Int64Range(0, 200_000).Draw(t, "b")) * 1_000
		if a > b {
			a, b = b, a
		}

		gotA, gotB := ResidentTaxOf(a, m), ResidentTaxOf(b, m)

		if gotA.Total() > gotB.Total() {
			t.Fatalf("taxable %d is taxed %d but the larger %d only %d", a, gotA.Total(), b, gotB.Total())
		}
	})
}

func TestResidentTaxShouldNeverBeNegative(t *testing.T) {
	tables := loadedMunicipalities(t)

	rapid.Check(t, func(t *rapid.T) {
		m := tables[rapid.IntRange(0, len(tables)-1).Draw(t, "municipality")]
		taxable := money.Yen(rapid.Int64Range(-10_000_000, 200_000_000).Draw(t, "taxable"))

		got := ResidentTaxOf(taxable, m)

		if got.Prefectural < 0 || got.Municipal < 0 || got.Total() < 0 {
			t.Fatalf("taxable %d gives a negative tax %+v", taxable, got)
		}
	})
}

func loadedMunicipalities(t *testing.T) []ResidentRates {
	t.Helper()

	names := municipalitiesWithTables(t)
	tables := make([]ResidentRates, 0, len(names))
	for _, name := range names {
		rates, _, _ := residentTablesAt(t, name, 2023)
		tables = append(tables, rates)
	}
	return tables
}

func TestOnlyThePerCapitaShouldMoveInTheYearItMoved(t *testing.T) {
	const moved = 2024

	for _, municipality := range municipalitiesWithTables(t) {
		t.Run(string(municipality), func(t *testing.T) {
			ratesBefore, levyBefore, exemptionBefore := residentTablesAt(t, municipality, moved-1)
			ratesAfter, levyAfter, exemptionAfter := residentTablesAt(t, municipality, moved)

			for _, c := range []struct {
				what      string
				same      bool
				shown     string
				wantMoved bool
			}{
				{"均等割", levyBefore == levyAfter, fmt.Sprintf("%+v → %+v", levyBefore, levyAfter), true},
				{"税率", ratesBefore == ratesAfter, fmt.Sprintf("%+v → %+v", ratesBefore, ratesAfter), false},
				{"非課税限度額", exemptionBefore == exemptionAfter,
					fmt.Sprintf("%+v → %+v", exemptionBefore, exemptionAfter), false},
			} {
				switch {
				case c.wantMoved && c.same:
					t.Fatalf("課税年度%d に%sが動いていない（%s）。この検査の前提である",
						moved, c.what, c.shown)
				case !c.wantMoved && !c.same:
					t.Errorf("均等割の改定に%sが引きずられている: 課税年度%d から %d へ %s",
						c.what, moved-1, moved, c.shown)
				}
			}

			if want := (ResidentExemption{PerPerson: 350_000, Addition: 210_000}); exemptionAfter != want {
				t.Errorf("非課税限度額 = %+v, want %+v", exemptionAfter, want)
			}
		})
	}
}

type spyT struct {
	failures []string
}

func (s *spyT) Helper() {}
func (s *spyT) Fatalf(format string, args ...any) {
	s.failures = append(s.failures, fmt.Sprintf(format, args...))
}

func TestMustAtShouldNameEveryTableWithNoRowForTheYear(t *testing.T) {
	tables := ResidentMunicipality{
		name:       "世田谷区",
		rates:      mustYearTable(t, YearRow[ResidentRates]{FromYear: 2018}),
		perCapita:  mustYearTable(t, YearRow[ResidentPerCapitaLevy]{FromYear: 2024}),
		exemptions: mustYearTable(t, YearRow[ResidentExemption]{FromYear: 2024}),
	}

	for name, c := range map[string]struct {
		year        date.Year
		wantNamed   []string
		wantUnnamed []string
	}{
		"行の無い 2 表を名指しし、ある 1 表には触れない": {
			year:        2023,
			wantNamed:   []string{ResidentPerCapitaTableName, ResidentExemptionTableName, "世田谷区"},
			wantUnnamed: []string{ResidentRateTableName},
		},
		"3 表とも行のある年は何も言わない": {year: 2024},
	} {
		t.Run(name, func(t *testing.T) {
			spy := &spyT{}

			tables.MustAt(spy, c.year)

			if len(c.wantNamed) == 0 {
				if len(spy.failures) != 0 {
					t.Fatalf("課税年度%d で %v と言っている", c.year, spy.failures)
				}
				return
			}
			if len(spy.failures) != 1 {
				t.Fatalf("報告が %d 件ある（1 件のはず）: %v", len(spy.failures), spy.failures)
			}
			for _, want := range c.wantNamed {
				if !strings.Contains(spy.failures[0], want) {
					t.Errorf("%s に触れていない: %q", want, spy.failures[0])
				}
			}
			for _, unwanted := range c.wantUnnamed {
				if strings.Contains(spy.failures[0], unwanted) {
					t.Errorf("行のある %s まで名指ししている: %q", unwanted, spy.failures[0])
				}
			}
		})
	}
}

func mustYearTable[V any](t *testing.T, rows ...YearRow[V]) YearTable[V] {
	t.Helper()

	built, err := NewYearTable(rows)
	if err != nil {
		t.Fatalf("NewYearTable: %v", err)
	}
	return built
}
