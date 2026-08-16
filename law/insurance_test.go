package law

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
)

const ordinaryYear = 2025

func TestLifeInsuranceDeductionShouldMatchTheSpreadsheet(t *testing.T) {
	table, err := sheets.New(os.DirFS("../testdata/sheets")).Table("life-insurance-deduction")
	if err != nil {
		t.Fatalf("Table: %v", err)
	}
	premiumColumn, ok := table.ColumnIndex("生命保険料")
	if !ok {
		t.Fatal("no 生命保険料 column")
	}
	deductionColumn, ok := table.ColumnIndex("生命保険料控除額")
	if !ok {
		t.Fatal("no 生命保険料控除額 column")
	}

	checked := 0
	for i, row := range table.Rows {
		premium, err := parseManYen(strings.TrimSuffix(row[premiumColumn], "万"))
		if err != nil {
			t.Fatalf("row %d: 生命保険料: %v", i+1, err)
		}
		want, err := parseManYen(row[deductionColumn])
		if err != nil {
			t.Fatalf("row %d: 生命保険料控除額: %v", i+1, err)
		}

		if got := LifeInsuranceDeduction(premium); got.IncomeTax != want {
			t.Errorf("保険料 %s: deduction %d, the spreadsheet says %d", row[premiumColumn], got.IncomeTax, want)
		}
		checked++
	}

	if checked < 75 {
		t.Errorf("only %d rows were checked; the golden table looks truncated", checked)
	}
}

func TestLifeInsuranceDeductionTotal(t *testing.T) {
	type testCase struct {
		General      money.Yen
		Medical      money.Yen
		Annuity      money.Yen
		WantIncome   money.Yen
		WantResident money.Yen
	}

	testCases := map[string]testCase{
		"one category only": {
			General: 80_000, WantIncome: 40_000, WantResident: 28_000,
		},
		"two categories add up": {
			General: 80_000, Medical: 80_000, WantIncome: 80_000, WantResident: 56_000,
		},
		"three categories are held to the smaller total cap": {
			General: 80_000, Medical: 80_000, Annuity: 80_000,
			WantIncome: 120_000, WantResident: 70_000,
		},
		"nothing at all": {
			WantIncome: 0, WantResident: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := LifeInsuranceDeductionTotal(tc.General, tc.Medical, tc.Annuity, ordinaryYear, false)

			if got.IncomeTax != tc.WantIncome {
				t.Errorf("IncomeTax = %d, want %d", got.IncomeTax, tc.WantIncome)
			}
			if got.Resident != tc.WantResident {
				t.Errorf("Resident = %d, want %d", got.Resident, tc.WantResident)
			}
		})
	}
}

func TestNoInsuranceDeductionShouldExceedItsCapOrThePremium(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		premium := money.Yen(rapid.Int64Range(0, 10_000_000).Draw(t, "premium"))

		life := LifeInsuranceDeduction(premium)
		quake := EarthquakeInsuranceDeduction(premium)

		if life.IncomeTax > LifeInsuranceCategoryCapIncomeTax || life.Resident > LifeInsuranceCategoryCapResident {
			t.Fatalf("premium %d deducts %+v, past a cap", premium, life)
		}
		if quake.IncomeTax > EarthquakeCapIncomeTax || quake.Resident > EarthquakeCapResident {
			t.Fatalf("premium %d deducts %+v, past a cap", premium, quake)
		}
		if life.IncomeTax > premium || quake.IncomeTax > premium {
			t.Fatalf("premium %d deducts more than was paid: life %+v, quake %+v", premium, life, quake)
		}
	})
}

func TestInsuranceDeductionsShouldRiseWithThePremium(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := money.Yen(rapid.Int64Range(0, 1_000_000).Draw(t, "a"))
		b := money.Yen(rapid.Int64Range(0, 1_000_000).Draw(t, "b"))
		if a > b {
			a, b = b, a
		}

		if LifeInsuranceDeduction(a).IncomeTax > LifeInsuranceDeduction(b).IncomeTax {
			t.Fatalf("premium %d deducts more than the larger %d", a, b)
		}
		if EarthquakeInsuranceDeduction(a).IncomeTax > EarthquakeInsuranceDeduction(b).IncomeTax {
			t.Fatalf("premium %d deducts more than the larger %d", a, b)
		}
	})
}

func TestTheTotalShouldNeverExceedTheSumOfItsParts(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		general := money.Yen(rapid.Int64Range(0, 500_000).Draw(t, "general"))
		medical := money.Yen(rapid.Int64Range(0, 500_000).Draw(t, "medical"))
		annuity := money.Yen(rapid.Int64Range(0, 500_000).Draw(t, "annuity"))

		total := LifeInsuranceDeductionTotal(general, medical, annuity, ordinaryYear, false)
		parts := LifeInsuranceDeduction(general).IncomeTax +
			LifeInsuranceDeduction(medical).IncomeTax +
			LifeInsuranceDeduction(annuity).IncomeTax

		if total.IncomeTax > parts {
			t.Fatalf("the total %d exceeds the sum of the parts %d", total.IncomeTax, parts)
		}
		if total.IncomeTax > LifeInsuranceTotalCapIncomeTax {
			t.Fatalf("the total %d exceeds the cap", total.IncomeTax)
		}
	})
}

func TestTheChildRearingLifeInsuranceDeductionShouldApplyOnlyInItsYears(t *testing.T) {
	cases := map[string]struct {
		premium money.Yen
		year    date.Year
		young   bool
		want    money.Yen
	}{
		"令和8年分・23歳未満あり: 上限 6 万円": {premium: 198_000, year: 2026, young: true, want: 60_000},
		"令和9年分まで延長された":           {premium: 198_000, year: 2027, young: true, want: 60_000},
		"令和10年分は元に戻る":            {premium: 198_000, year: 2028, young: true, want: 40_000},
		"令和7年分にはまだ無い":            {premium: 198_000, year: 2025, young: true, want: 40_000},
		"23歳未満の扶養親族がいなければ適用外":    {premium: 198_000, year: 2026, young: false, want: 40_000},

		"令和8年分: 30,000 は全額":     {premium: 30_000, year: 2026, young: true, want: 30_000},
		"改正前: 30,000 は 1/2+1 万": {premium: 30_000, year: 2025, young: true, want: 25_000},
		"令和8年分: 60,000":         {premium: 60_000, year: 2026, young: true, want: 45_000},
		"改正前: 60,000":           {premium: 60_000, year: 2025, young: true, want: 35_000},
		"令和8年分: 120,000":        {premium: 120_000, year: 2026, young: true, want: 60_000},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := GeneralLifeInsuranceDeduction(c.premium, c.year, c.young)
			if got.IncomeTax != c.want {
				t.Errorf("所得税の控除が %v である（%v のはず）", got.IncomeTax, c.want)
			}
		})
	}
}

func TestTheChildRearingSpecialShouldNotTouchTheResidentDeduction(t *testing.T) {
	for _, year := range []int{2025, 2026, 2027, 2028} {
		got := GeneralLifeInsuranceDeduction(198_000, date.Year(year), true)
		if got.Resident != LifeInsuranceCategoryCapResident {
			t.Errorf("%d 年の住民税の控除が %v である（%v のはず）",
				year, got.Resident, LifeInsuranceCategoryCapResident)
		}
	}
}
