package law

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
	"github.com/Kuniwak/lifeplan/tsv"
)

const middleBandCeiling money.Yen = 480_000

func TestSpouseDeduction(t *testing.T) {
	type testCase struct {
		SpouseIncome    money.Yen
		TaxpayerIncome  money.Yen
		SpouseAge       int
		WantIncomeTax   money.Yen
		WantResidentTax money.Yen
	}

	testCases := map[string]testCase{
		"an ordinary taxpayer and a spouse with no income": {
			SpouseIncome: 0, TaxpayerIncome: 5_000_000, SpouseAge: 40,
			WantIncomeTax: 380_000, WantResidentTax: 330_000,
		},
		"the spouse earns exactly the ceiling (boundary value)": {
			SpouseIncome: 480_000, TaxpayerIncome: 5_000_000, SpouseAge: 40,
			WantIncomeTax: 380_000, WantResidentTax: 330_000,
		},
		"one yen past the ceiling leaves nothing here (boundary)": {
			SpouseIncome: 480_001, TaxpayerIncome: 5_000_000, SpouseAge: 40,
			WantIncomeTax: 0, WantResidentTax: 0,
		},
		"an elderly spouse brings more": {
			SpouseIncome: 0, TaxpayerIncome: 5_000_000, SpouseAge: 70,
			WantIncomeTax: 480_000, WantResidentTax: 380_000,
		},
		"one year short of elderly (boundary value)": {
			SpouseIncome: 0, TaxpayerIncome: 5_000_000, SpouseAge: 69,
			WantIncomeTax: 380_000, WantResidentTax: 330_000,
		},
		"exactly nine million of taxpayer income (boundary value)": {
			SpouseIncome: 0, TaxpayerIncome: 9_000_000, SpouseAge: 40,
			WantIncomeTax: 380_000, WantResidentTax: 330_000,
		},
		"one yen past nine million (boundary value)": {
			SpouseIncome: 0, TaxpayerIncome: 9_000_001, SpouseAge: 40,
			WantIncomeTax: 260_000, WantResidentTax: 220_000,
		},
		"the third band": {
			SpouseIncome: 0, TaxpayerIncome: 9_800_000, SpouseAge: 40,
			WantIncomeTax: 130_000, WantResidentTax: 110_000,
		},
		"exactly ten million (boundary value)": {
			SpouseIncome: 0, TaxpayerIncome: 10_000_000, SpouseAge: 40,
			WantIncomeTax: 130_000, WantResidentTax: 110_000,
		},
		"past ten million leaves nothing (boundary value)": {
			SpouseIncome: 0, TaxpayerIncome: 10_000_001, SpouseAge: 40,
			WantIncomeTax: 0, WantResidentTax: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := SpouseDeduction(tc.SpouseIncome, tc.TaxpayerIncome, tc.SpouseAge, middleBandCeiling, 2020)

			if got.IncomeTax != tc.WantIncomeTax {
				t.Errorf("IncomeTax = %d, want %d", got.IncomeTax, tc.WantIncomeTax)
			}
			if got.Resident != tc.WantResidentTax {
				t.Errorf("Resident = %d, want %d", got.Resident, tc.WantResidentTax)
			}
		})
	}
}

func TestSpouseSpecialDeductionShouldMatchTheSpreadsheet(t *testing.T) {
	table, err := sheets.New(os.DirFS("../testdata/sheets")).Table("spouse-special-deduction")
	if err != nil {
		t.Fatalf("Table: %v", err)
	}

	tiers := []money.Yen{5_000_000, 9_200_000, 9_800_000}
	columns := []tsv.ColumnName{"納税者本人の合計所得金額", "col3", "col4"}

	spouseColumn, ok := table.ColumnIndex("配偶者の合計所得金額")
	if !ok {
		t.Fatal("no 配偶者の合計所得金額 column")
	}

	checked := 0
	for i, row := range table.Rows {
		if i == 0 {
			continue
		}

		lower, err := parseManYen(row[spouseColumn])
		if err != nil {
			t.Fatalf("row %d: %v", i+1, err)
		}
		if i+1 >= len(table.Rows) {
			continue
		}
		upper, err := parseManYen(table.Rows[i+1][spouseColumn])
		if err != nil {
			t.Fatalf("row %d: %v", i+2, err)
		}
		if upper > SpouseSpecialIncomeCeilingAt(2020) {
			continue
		}

		for tier, column := range columns {
			index, ok := table.ColumnIndex(column)
			if !ok {
				t.Fatalf("no %s column", column)
			}
			cell := row[index]
			if cell == "#N/A" {
				continue
			}
			want, err := parseManYen(cell)
			if err != nil {
				t.Fatalf("row %d, %s: %v", i+1, column, err)
			}

			if got := SpouseSpecialDeduction(upper, tiers[tier], middleBandCeiling, 2020); got.IncomeTax != want {
				t.Errorf("配偶者所得 %s万超 %s万以下, taxpayer tier %d: deduction %d, the spreadsheet says %d",
					row[spouseColumn], table.Rows[i+1][spouseColumn], tier, got.IncomeTax, want)
			}
			checked++
			_ = lower
		}
	}

	if checked < 20 {
		t.Errorf("only %d cells were checked; the golden table looks truncated", checked)
	}
}

func TestTheSpouseSpecialDeductionShouldStopAtTheStatutoryCeiling(t *testing.T) {
	type testCase struct {
		SpouseIncome money.Yen
		Expected     money.Yen
	}

	testCases := map[string]testCase{
		"exactly the ceiling (boundary value)": {SpouseIncome: 1_330_000, Expected: 30_000},
		"one yen past the ceiling (boundary)":  {SpouseIncome: 1_330_001, Expected: 0},
		"where the spreadsheet still deducts":  {SpouseIncome: 1_335_000, Expected: 0},
		"well past the ceiling":                {SpouseIncome: 2_000_000, Expected: 0},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := SpouseSpecialDeduction(tc.SpouseIncome, 5_000_000, middleBandCeiling, 2020)

			if got.IncomeTax != tc.Expected {
				t.Errorf("SpouseSpecialDeduction(%d) = %d, want %d", tc.SpouseIncome, got.IncomeTax, tc.Expected)
			}
		})
	}
}

func TestTheOtherHumanDeductionGapsShouldEqualTheDifference(t *testing.T) {
	gap := func(d Deduction) money.Yen { return d.IncomeTax - d.Resident }

	cases := map[string]struct {
		got  money.Yen
		want money.Yen
	}{
		"一般の扶養親族 5万": {gap(DependentDeduction(16, false)), 50_000},
		"特定扶養親族 18万": {gap(DependentDeduction(19, false)), 180_000},
		"老人扶養親族 10万": {gap(DependentDeduction(70, false)), 100_000},
		"同居老親等 13万":  {gap(DependentDeduction(70, true)), 130_000},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if c.got != c.want {
				t.Errorf("引き算で出した差は %d、条文の差額表は %d", c.got, c.want)
			}
		})
	}
}

func TestTheTwoSpouseDeductionsShouldNeverBothApply(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		spouseIncome := money.Yen(rapid.Int64Range(0, 3_000_000).Draw(t, "spouseIncome"))
		taxpayerIncome := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "taxpayerIncome"))
		spouseAge := rapid.IntRange(20, 100).Draw(t, "spouseAge")

		ceiling := rapid.SampledFrom([]money.Yen{380_000, 480_000, 580_000}).Draw(t, "ceiling")
		incomeYear := rapid.SampledFrom([]int{2019, 2020, 2025}).Draw(t, "所得の年")

		ordinary := SpouseDeduction(spouseIncome, taxpayerIncome, spouseAge, ceiling, date.Year(incomeYear))
		special := SpouseSpecialDeduction(spouseIncome, taxpayerIncome, ceiling, date.Year(incomeYear))

		if ordinary != (Deduction{}) && special != (Deduction{}) {
			t.Fatalf("spouse income %d and taxpayer income %d bring both %+v and %+v",
				spouseIncome, taxpayerIncome, ordinary, special)
		}

		total := SpouseDeductionTotal(spouseIncome, taxpayerIncome, spouseAge, ceiling, date.Year(incomeYear))
		if want := max(ordinary.IncomeTax, special.IncomeTax); total.IncomeTax != want {
			t.Fatalf("the total %d is neither of the two deductions %d and %d",
				total.IncomeTax, ordinary.IncomeTax, special.IncomeTax)
		}
	})
}

func TestASpouseEarningMoreShouldNeverDeductMore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := money.Yen(rapid.Int64Range(0, 3_000_000).Draw(t, "a"))
		b := money.Yen(rapid.Int64Range(0, 3_000_000).Draw(t, "b"))
		if a > b {
			a, b = b, a
		}
		taxpayerIncome := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "taxpayerIncome"))
		ceiling := rapid.SampledFrom([]money.Yen{380_000, 480_000, 580_000}).Draw(t, "ceiling")
		incomeYear := rapid.SampledFrom([]int{2019, 2020, 2025}).Draw(t, "所得の年")

		gotA := SpouseDeductionTotal(a, taxpayerIncome, 40, ceiling, date.Year(incomeYear))
		gotB := SpouseDeductionTotal(b, taxpayerIncome, 40, ceiling, date.Year(incomeYear))

		if gotB.IncomeTax > gotA.IncomeTax {
			t.Fatalf("a spouse earning %d deducts %d but one earning %d deducts %d",
				a, gotA.IncomeTax, b, gotB.IncomeTax)
		}
	})
}

func TestDependentDeduction(t *testing.T) {
	type testCase struct {
		Age             int
		LivingWith      bool
		WantIncomeTax   money.Yen
		WantResidentTax money.Yen
	}

	testCases := map[string]testCase{
		"a small child brings nothing (the allowance does instead)": {Age: 5, WantIncomeTax: 0, WantResidentTax: 0},
		"one year short of sixteen (boundary value)":                {Age: 15, WantIncomeTax: 0, WantResidentTax: 0},
		"sixteen (boundary value)":                                  {Age: 16, WantIncomeTax: 380_000, WantResidentTax: 330_000},
		"eighteen, still general (boundary value)":                  {Age: 18, WantIncomeTax: 380_000, WantResidentTax: 330_000},
		"nineteen, the university years begin (boundary value)":     {Age: 19, WantIncomeTax: 630_000, WantResidentTax: 450_000},
		"twenty two, the last of them (boundary value)":             {Age: 22, WantIncomeTax: 630_000, WantResidentTax: 450_000},
		"twenty three, general again (boundary value)":              {Age: 23, WantIncomeTax: 380_000, WantResidentTax: 330_000},
		"an elderly dependant living elsewhere (boundary value)":    {Age: 70, WantIncomeTax: 480_000, WantResidentTax: 380_000},
		"an elderly dependant living with the taxpayer":             {Age: 70, LivingWith: true, WantIncomeTax: 580_000, WantResidentTax: 450_000},
		"one year short of elderly (boundary value)":                {Age: 69, WantIncomeTax: 380_000, WantResidentTax: 330_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := DependentDeduction(tc.Age, tc.LivingWith)

			if got.IncomeTax != tc.WantIncomeTax {
				t.Errorf("IncomeTax = %d, want %d", got.IncomeTax, tc.WantIncomeTax)
			}
			if got.Resident != tc.WantResidentTax {
				t.Errorf("Resident = %d, want %d", got.Resident, tc.WantResidentTax)
			}
		})
	}
}

func TestNoDeductionShouldEverBeNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		spouseIncome := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "spouseIncome"))
		taxpayerIncome := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "taxpayerIncome"))
		age := rapid.IntRange(0, 110).Draw(t, "age")
		ceiling := rapid.SampledFrom([]money.Yen{380_000, 480_000, 580_000}).Draw(t, "ceiling")
		incomeYear := rapid.SampledFrom([]int{2019, 2020, 2025}).Draw(t, "所得の年")

		spouse := SpouseDeductionTotal(spouseIncome, taxpayerIncome, age, ceiling, date.Year(incomeYear))
		dependent := DependentDeduction(age, rapid.Bool().Draw(t, "livingWith"))

		for name, d := range map[string]Deduction{"spouse": spouse, "dependent": dependent} {
			if d.IncomeTax < 0 || d.Resident < 0 {
				t.Fatalf("the %s deduction is negative: %+v", name, d)
			}
			if d.Resident > d.IncomeTax {
				t.Fatalf("the %s deduction is larger for the resident tax than the income tax: %+v", name, d)
			}
		}
	})
}

func TestSpouseDeductionsOfShouldRefuseAYearBeforeTheRecord(t *testing.T) {
	const ceiling money.Yen = 380_000

	for _, branch := range []struct {
		name                         string
		spouseIncome, taxpayerIncome money.Yen
	}{
		{name: "控除対象配偶者", spouseIncome: 0, taxpayerIncome: 5_000_000},
		{name: "配偶者特別控除", spouseIncome: 600_000, taxpayerIncome: 5_000_000},
		{name: "本人が千万円超でどちらも無い", spouseIncome: 0, taxpayerIncome: 12_000_000},
	} {
		t.Run(branch.name, func(t *testing.T) {
			for _, c := range []struct {
				incomeYear date.Year
				wantPanic  bool
			}{
				{incomeYear: 2018, wantPanic: false},
				{incomeYear: 2017, wantPanic: true},
				{incomeYear: 2016, wantPanic: true},
			} {
				panicked := panictest.Recovered(func() {
					SpouseDeductionsOf(branch.spouseIncome, branch.taxpayerIncome, 40, ceiling, c.incomeYear)
				}) != nil

				if panicked != c.wantPanic {
					t.Errorf("所得の年%d で panic が %v である（%v のはず）", c.incomeYear, panicked, c.wantPanic)
				}
			}
		})
	}
}
