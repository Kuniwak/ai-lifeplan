package law

import (
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
)

func TestSalaryIncomeDeduction(t *testing.T) {
	type testCase struct {
		Salary   money.Yen
		Expected money.Yen
	}

	testCases := map[string]testCase{
		"no salary at all (the table still states 55万)": {Salary: 0, Expected: 550_000},
		"one yen (lower boundary value)":                {Salary: 1, Expected: 550_000},
		"top of the flat band (boundary value)":         {Salary: 1_625_000, Expected: 550_000},
		"one yen into the 40% band (boundary)":          {Salary: 1_625_000, Expected: 550_000},
		"top of the 40% band (boundary value)":          {Salary: 1_800_000, Expected: 620_000},
		"in the 30% band (representative)":              {Salary: 3_200_000, Expected: 1_040_000},
		"top of the 30% band (boundary value)":          {Salary: 3_600_000, Expected: 1_160_000},
		"in the 20% band (representative)":              {Salary: 4_800_000, Expected: 1_400_000},
		"top of the 20% band (boundary value)":          {Salary: 6_600_000, Expected: 1_760_000},
		"in the 10% band (representative)":              {Salary: 8_000_000, Expected: 1_900_000},
		"top of the 10% band (boundary value)":          {Salary: 8_500_000, Expected: 1_950_000},
		"one yen above the cap (boundary value)":        {Salary: 8_500_000, Expected: 1_950_000},
		"far above the cap":                             {Salary: 14_000_000, Expected: 1_950_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := SalaryIncomeDeduction(tc.Salary, 2024)

			if got != tc.Expected {
				t.Errorf("SalaryIncomeDeduction(%d, 2024) = %d, want %d", tc.Salary, got, tc.Expected)
			}
		})
	}
}

func TestSalaryIncomeShouldMatchTheSpreadsheet(t *testing.T) {
	table, err := sheets.New(os.DirFS("../testdata/sheets")).Table("salary-income-deduction")
	if err != nil {
		t.Fatalf("Table: %v", err)
	}

	salaryColumn, ok := table.ColumnIndex("給与収入額")
	if !ok {
		t.Fatal("no 給与収入額 column")
	}
	deductionColumn, ok := table.ColumnIndex("給与所得控除")
	if !ok {
		t.Fatal("no 給与所得控除 column")
	}
	incomeColumn, ok := table.ColumnIndex("給与所得額")
	if !ok {
		t.Fatal("no 給与所得額 column")
	}

	checked := 0
	for i, row := range table.Rows {
		salary, err := parseManYen(row[salaryColumn])
		if err != nil {
			t.Fatalf("row %d: 給与収入額: %v", i+1, err)
		}
		wantDeduction, err := parseManYen(row[deductionColumn])
		if err != nil {
			t.Fatalf("row %d: 給与所得控除: %v", i+1, err)
		}
		wantIncome, err := parseManYen(row[incomeColumn])
		if err != nil {
			t.Fatalf("row %d: 給与所得額: %v", i+1, err)
		}

		if got := SalaryIncomeDeduction(salary, 2024); got != wantDeduction {
			t.Errorf("給与収入 %s万: deduction %d, the spreadsheet says %d", row[salaryColumn], got, wantDeduction)
		}
		if got := SalaryIncome(salary, 2024); got != wantIncome {
			t.Errorf("給与収入 %s万: income %d, the spreadsheet says %d", row[salaryColumn], got, wantIncome)
		}
		checked++
	}

	if checked < 600 {
		t.Errorf("only %d rows were checked; the golden table looks truncated", checked)
	}
}

func TestSalaryIncomeDeductionOfReiwaOne(t *testing.T) {
	type testCase struct {
		Salary   money.Yen
		Expected money.Yen
	}

	testCases := map[string]testCase{
		"収入なし（下限が立つ）":         {Salary: 0, Expected: 650_000},
		"1 円（境界値）":            {Salary: 1, Expected: 650_000},
		"一律の帯の上端 162.5万（境界値）": {Salary: 1_625_000, Expected: 650_000},
		"40% の帯に 1 円入る（境界値）":  {Salary: 1_625_000, Expected: 650_000},
		"40% の帯の上端 180万（境界値）": {Salary: 1_800_000, Expected: 720_000},

		"30% の帯（代表値 320万）":    {Salary: 3_200_000, Expected: 1_140_000},
		"30% の帯の上端 360万（境界値）": {Salary: 3_600_000, Expected: 1_260_000},

		"20% の帯（代表値 480万）":    {Salary: 4_800_000, Expected: 1_500_000},
		"20% の帯の上端 660万（境界値）": {Salary: 6_600_000, Expected: 1_860_000},

		"10% の帯（代表値 800万）":      {Salary: 8_000_000, Expected: 2_000_000},
		"令和2年分なら上限の 850万（境界値）":  {Salary: 8_500_000, Expected: 2_050_000},
		"10% の帯の上端 1,000万（境界値）": {Salary: 10_000_000, Expected: 2_200_000},

		"上限に 1 円入る（境界値）": {Salary: 10_000_001, Expected: 2_200_000},
		"上限のはるか上":        {Salary: 14_000_000, Expected: 2_200_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := SalaryIncomeDeduction(tc.Salary, 2019)

			if got != tc.Expected {
				t.Errorf("SalaryIncomeDeduction(%d, 2019) = %d, want %d", tc.Salary, got, tc.Expected)
			}
		})
	}
}

func TestSalaryIncomeDeductionShouldNeverExceedTheSalary(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		salary := money.Yen(rapid.Int64Range(0, 100_000_000).Draw(t, "salary"))
		year := date.Year(rapid.IntRange(2015, 2100).Draw(t, "year"))

		floor, ceiling := SalaryDeductionBoundsAt(year)

		got := SalaryIncomeDeduction(salary, year)

		if got > ceiling {
			t.Fatalf("%d年: the deduction %d exceeds the statutory ceiling %d", year, got, ceiling)
		}
		if salary > 0 && got < floor {
			t.Fatalf("%d年: the deduction %d is below the statutory floor %d for a salary of %d", year, got, floor, salary)
		}
	})
}

func TestSalaryIncomeShouldRiseWithTheSalary(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := money.Yen(rapid.Int64Range(0, 100_000_000).Draw(t, "a"))
		b := money.Yen(rapid.Int64Range(0, 100_000_000).Draw(t, "b"))
		if a > b {
			a, b = b, a
		}
		year := date.Year(rapid.IntRange(2015, 2100).Draw(t, "year"))

		gotA, gotB := SalaryIncome(a, year), SalaryIncome(b, year)

		if gotA > gotB {
			t.Fatalf("salary %d gives income %d but the larger salary %d gives only %d", a, gotA, b, gotB)
		}
	})
}

func TestSalaryIncomeShouldNeverExceedTheSalary(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		salary := money.Yen(rapid.Int64Range(0, 100_000_000).Draw(t, "salary"))
		year := date.Year(rapid.IntRange(2015, 2100).Draw(t, "year"))

		got := SalaryIncome(salary, year)

		if got > salary {
			t.Fatalf("salary %d gives income %d, which is more than was earned", salary, got)
		}
		if got < 0 {
			t.Fatalf("salary %d gives a negative income %d", salary, got)
		}
	})
}
