package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"os"
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

func healthGradeTable(t *testing.T) StandardRemunerationTable {
	t.Helper()
	return loadStandardRemuneration(t, StandardRemunerationHealthTableName)
}

func pensionGradeTable(t *testing.T) StandardRemunerationTable {
	t.Helper()
	return loadStandardRemuneration(t, StandardRemunerationPensionTableName)
}

func loadStandardRemuneration(t *testing.T, name string) StandardRemunerationTable {
	t.Helper()

	return MustLoadStandardRemunerations(t, os.DirFS("../"+LawDirectory), name)
}

func TestStandardRemunerationHealth(t *testing.T) {
	type testCase struct {
		MonthlyPay money.Yen
		Expected   money.Yen
	}

	testCases := map[string]testCase{
		"no pay at all falls in the lowest grade":          {MonthlyPay: 0, Expected: 58_000},
		"one yen below the first boundary (boundary)":      {MonthlyPay: 62_999, Expected: 58_000},
		"exactly on the first boundary (boundary)":         {MonthlyPay: 63_000, Expected: 68_000},
		"one yen below a boundary":                         {MonthlyPay: 72_999, Expected: 68_000},
		"exactly on a boundary belongs to the grade above": {MonthlyPay: 73_000, Expected: 78_000},
		"a pay in the middle of a grade":                   {MonthlyPay: 586_000, Expected: 590_000},
		"one yen below the top grade (boundary)":           {MonthlyPay: 1_354_999, Expected: 1_330_000},
		"exactly at the top grade (boundary)":              {MonthlyPay: 1_355_000, Expected: 1_390_000},
		"far above the top grade":                          {MonthlyPay: 99_000_000, Expected: 1_390_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := healthGradeTable(t).Lookup(tc.MonthlyPay)

			if got != tc.Expected {
				t.Errorf("健康保険の標準報酬月額表: 報酬月額 %d は %d のはずだが %d になった", tc.MonthlyPay, tc.Expected, got)
			}
		})
	}
}

func TestStandardRemunerationPension(t *testing.T) {
	type testCase struct {
		MonthlyPay money.Yen
		Expected   money.Yen
	}

	testCases := map[string]testCase{
		"no pay at all falls in the lowest grade":          {MonthlyPay: 0, Expected: 88_000},
		"one yen below a boundary":                         {MonthlyPay: 92_999, Expected: 88_000},
		"exactly on a boundary belongs to the grade above": {MonthlyPay: 93_000, Expected: 98_000},
		"one yen below the top grade (boundary)":           {MonthlyPay: 634_999, Expected: 620_000},
		"exactly at the top grade (boundary)":              {MonthlyPay: 635_000, Expected: 650_000},
		"far above the top grade":                          {MonthlyPay: 99_000_000, Expected: 650_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := pensionGradeTable(t).Lookup(tc.MonthlyPay)

			if got != tc.Expected {
				t.Errorf("厚生年金の標準報酬月額表: 報酬月額 %d は %d のはずだが %d になった", tc.MonthlyPay, tc.Expected, got)
			}
		})
	}
}

func TestTheTwoGradeTablesShouldAgreeWhereTheyOverlap(t *testing.T) {
	health := healthGradeTable(t)
	pension := pensionGradeTable(t)

	healthLower := make(map[money.Yen]money.Yen, len(health.Bands()))
	for _, band := range health.Bands() {
		healthLower[band.Standard] = band.LowerBound
	}

	checked := 0
	for _, band := range pension.Bands()[1:] {
		lower, ok := healthLower[band.Standard]
		if !ok {
			t.Errorf("厚生年金 has a standard remuneration of %d that 健康保険 does not", band.Standard)
			continue
		}
		if lower != band.LowerBound {
			t.Errorf("standard remuneration %d starts at %d in 健康保険 but at %d in 厚生年金",
				band.Standard, lower, band.LowerBound)
		}
		checked++
	}

	if checked != len(pension.Bands())-1 {
		t.Fatalf("only %d of %d 厚生年金 grades were compared", checked, len(pension.Bands())-1)
	}
	t.Logf("compared %d grades", checked)
}

func TestStandardRemunerationShouldNeverFallAsPayRises(t *testing.T) {
	health := healthGradeTable(t)
	pension := pensionGradeTable(t)

	rapid.Check(t, func(t *rapid.T) {
		lower := money.Yen(rapid.Int64Range(0, 3_000_000).Draw(t, "lower"))
		higher := money.Yen(rapid.Int64Range(0, 3_000_000).Draw(t, "higher"))
		if lower > higher {
			lower, higher = higher, lower
		}

		if health.Lookup(lower) > health.Lookup(higher) {
			t.Fatalf("健康保険: pay %d gives %d but pay %d gives less, %d",
				lower, health.Lookup(lower), higher, health.Lookup(higher))
		}
		if pension.Lookup(lower) > pension.Lookup(higher) {
			t.Fatalf("厚生年金: pay %d gives %d but pay %d gives less, %d",
				lower, pension.Lookup(lower), higher, pension.Lookup(higher))
		}
	})
}

func TestStandardRemunerationShouldAlwaysBeAValueFromTheTable(t *testing.T) {
	health := healthGradeTable(t)
	pension := pensionGradeTable(t)

	rapid.Check(t, func(t *rapid.T) {
		pay := money.Yen(rapid.Int64Range(0, 100_000_000).Draw(t, "pay"))

		healthStandard := health.Lookup(pay)
		pensionStandard := pension.Lookup(pay)

		if healthStandard%StandardBonusUnit != 0 || pensionStandard%StandardBonusUnit != 0 {
			t.Fatalf("pay %d gives %d and %d, which are not whole thousands", pay, healthStandard, pensionStandard)
		}
		grades := pension.Bands()
		floor := grades[0].Standard
		ceiling := grades[len(grades)-1].Standard
		if want := min(max(healthStandard, floor), ceiling); pensionStandard != want {
			t.Fatalf("pay %d gives a 健康保険 standard of %d, so 厚生年金 should be %d, but it is %d",
				pay, healthStandard, want, pensionStandard)
		}
	})
}

func TestGradeBoundariesShouldMatchThePremiumTable(t *testing.T) {
	health := healthGradeTable(t)

	type row struct {
		Lower    money.Yen
		Upper    money.Yen
		Standard money.Yen
	}

	rows := []row{
		{Lower: 0, Upper: 63_000, Standard: 58_000},
		{Lower: 63_000, Upper: 73_000, Standard: 68_000},
		{Lower: 93_000, Upper: 101_000, Standard: 98_000},
		{Lower: 195_000, Upper: 210_000, Standard: 200_000},
		{Lower: 395_000, Upper: 425_000, Standard: 410_000},
		{Lower: 575_000, Upper: 605_000, Standard: 590_000},
		{Lower: 605_000, Upper: 635_000, Standard: 620_000},
		{Lower: 635_000, Upper: 665_000, Standard: 650_000},
		{Lower: 695_000, Upper: 730_000, Standard: 710_000},
		{Lower: 810_000, Upper: 855_000, Standard: 830_000},
		{Lower: 1_295_000, Upper: 1_355_000, Standard: 1_330_000},
		{Lower: 1_355_000, Upper: 0, Standard: 1_390_000},
	}

	for _, r := range rows {
		if got := health.Lookup(r.Lower); got != r.Standard {
			t.Errorf("報酬月額 %d は %d 円の等級のはずだが %d になった", r.Lower, r.Standard, got)
		}
		if r.Upper == 0 {
			continue
		}
		if got := health.Lookup(r.Upper - 1); got != r.Standard {
			t.Errorf("報酬月額 %d は %d 円の等級のはずだが %d になった", r.Upper-1, r.Standard, got)
		}
		if got := health.Lookup(r.Upper); got == r.Standard {
			t.Errorf("報酬月額 %d は %d 円の等級に入ってはならない（円未満）", r.Upper, r.Standard)
		}
	}
}

func TestPensionStandardRemunerationCeiling(t *testing.T) {
	type testCase struct {
		Year     int
		Month    int
		Expected money.Yen
	}

	testCases := map[string]testCase{
		"before the rise":                        {Year: 2018, Month: 4, Expected: 620_000},
		"the last month of the old ceiling":      {Year: 2020, Month: 8, Expected: 620_000},
		"the month the ceiling rises (boundary)": {Year: 2020, Month: 9, Expected: 650_000},
		"after the rise":                         {Year: 2022, Month: 12, Expected: 650_000},
		"far in the future, unchanged so far":    {Year: 2094, Month: 1, Expected: 650_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := PensionStandardRemunerationCeiling(date.Year(tc.Year), tc.Month)

			if got != tc.Expected {
				t.Errorf("PensionStandardRemunerationCeiling(%d, %d) = %d, want %d",
					tc.Year, tc.Month, got, tc.Expected)
			}
		})
	}
}

func TestPensionGradeBoundariesShouldMatchTheHistoryTable(t *testing.T) {
	pension := pensionGradeTable(t)

	type row struct {
		Lower    money.Yen
		Upper    money.Yen
		Standard money.Yen
	}

	rows := []row{
		{Lower: 0, Upper: 93_000, Standard: 88_000},
		{Lower: 93_000, Upper: 101_000, Standard: 98_000},
		{Lower: 101_000, Upper: 107_000, Standard: 104_000},
		{Lower: 195_000, Upper: 210_000, Standard: 200_000},
		{Lower: 370_000, Upper: 395_000, Standard: 380_000},
		{Lower: 395_000, Upper: 425_000, Standard: 410_000},
		{Lower: 575_000, Upper: 605_000, Standard: 590_000},
		{Lower: 605_000, Upper: 635_000, Standard: 620_000},
		{Lower: 635_000, Upper: 0, Standard: 650_000},
	}

	const grades = 32

	if len(pension.Bands()) != grades {
		t.Errorf("the 厚生年金 table has %d grades, but the history table has %d",
			len(pension.Bands()), grades)
	}

	for _, r := range rows {
		if got := pension.Lookup(r.Lower); got != r.Standard {
			t.Errorf("報酬月額 %d は %d 円の等級のはずだが %d になった", r.Lower, r.Standard, got)
		}
		if r.Upper == 0 {
			continue
		}
		if got := pension.Lookup(r.Upper - 1); got != r.Standard {
			t.Errorf("報酬月額 %d は %d 円の等級のはずだが %d になった", r.Upper-1, r.Standard, got)
		}
		if got := pension.Lookup(r.Upper); got == r.Standard {
			t.Errorf("報酬月額 %d は %d 円の等級に入ってはならない（円未満）", r.Upper, r.Standard)
		}
	}
}

func TestTheCeilingHistoryShouldEndAtTheTopGradeOfTheTable(t *testing.T) {
	pension := pensionGradeTable(t)
	grades := pension.Bands()

	top := grades[len(grades)-1].Standard
	current := PensionStandardRemunerationCeiling(date.Year(9999), 12)

	if current != top {
		t.Errorf("the ceiling history ends at %d but the 厚生年金 table stops at %d; one of the two was revised without the other",
			current, top)
	}
}

func TestParseStandardRemunerationTableNG(t *testing.T) {
	type testCase struct {
		Table *tsv.Table
	}

	testCases := map[string]testCase{
		"no lower bound column": {Table: &tsv.Table{
			Header: []tsv.ColumnName{"等級", StandardRemunerationValueColumn},
			Rows:   [][]string{{"1", "58000"}},
		}},
		"no standard remuneration column": {Table: &tsv.Table{
			Header: []tsv.ColumnName{"等級", StandardRemunerationLowerColumn},
			Rows:   [][]string{{"1", "0"}},
		}},
		"no grades at all": {Table: &tsv.Table{
			Header: []tsv.ColumnName{StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
		}},
		"the lowest grade does not start at zero": {Table: &tsv.Table{
			Header: []tsv.ColumnName{StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
			Rows:   [][]string{{"63000", "68000"}, {"73000", "78000"}},
		}},
		"a lower bound that is not an amount": {Table: &tsv.Table{
			Header: []tsv.ColumnName{StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
			Rows:   [][]string{{"", "58000"}},
		}},
		"a standard remuneration that is not an amount": {Table: &tsv.Table{
			Header: []tsv.ColumnName{StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
			Rows:   [][]string{{"0", "五万八千"}},
		}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := ParseStandardRemunerationTable(tc.Table)

			if err == nil {
				t.Fatal("ParseStandardRemunerationTable succeeded; a table that cannot be looked up safely must be refused")
			}
		})
	}
}

func TestAStandardRemunerationTableShouldNotMixTwoVersions(t *testing.T) {
	type testCase struct {
		Table       *tsv.Table
		WantRefused bool
	}

	testCases := map[string]testCase{
		"一つの版": {Table: &tsv.Table{
			Header: []tsv.ColumnName{LawStartYearColumn, StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
			Rows:   [][]string{{"2020", "0", "58000"}, {"2020", "63000", "68000"}},
		}},
		"二つの版が混ざっている": {Table: &tsv.Table{
			Header: []tsv.ColumnName{LawStartYearColumn, StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
			Rows: [][]string{
				{"2016", "0", "58000"}, {"2016", "63000", "68000"},
				{"2020", "0", "58000"}, {"2020", "63000", "68000"},
			},
		}, WantRefused: true},
		"二つの版が混ざり、年が全部 不明": {Table: &tsv.Table{
			Header: []tsv.ColumnName{LawStartYearColumn, StandardRemunerationLowerColumn, StandardRemunerationValueColumn},
			Rows: [][]string{
				{string(validate.Unknown), "0", "58000"}, {string(validate.Unknown), "63000", "68000"},
				{string(validate.Unknown), "0", "58000"}, {string(validate.Unknown), "63000", "68000"},
			},
		}, WantRefused: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := ParseStandardRemunerationTable(tc.Table)

			if tc.WantRefused {
				if err == nil {
					t.Fatal("二つの版が混ざった表が受け付けられた")
				}
				if !strings.Contains(err.Error(), "0") {
					t.Errorf("エラーが重なった等級を言っていない: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("通るはずの表が拒まれた: %v", err)
			}
		})
	}
}

func TestThePensionGradeTableShouldMatchTheVerificationBlocksExample(t *testing.T) {
	got := pensionGradeTable(t).Lookup(107_000)
	if want := money.Yen(110_000); got != want {
		t.Errorf("107,000円の標準報酬月額 = %d, want %d", got, want)
	}
}

func TestTheTwoLagsStepInDifferentMonths(t *testing.T) {
	for _, c := range []struct {
		month                  int
		wantDecision, wantRate date.Year
	}{
		{1, 2023, 2023},
		{3, 2023, 2023},
		{4, 2023, 2024},
		{9, 2023, 2024},
		{10, 2024, 2024},
		{12, 2024, 2024},
	} {
		t.Run(strconv.Itoa(c.month), func(t *testing.T) {
			if got := RegularDecisionYearOnPayslip(2024, c.month); got != c.wantDecision {
				t.Errorf("2024/%02d の定時決定の年 = %d, want %d", c.month, got, c.wantDecision)
			}
			if got := RateFiscalYearOnPayslip(2024, c.month); got != c.wantRate {
				t.Errorf("2024/%02d の料率の年度 = %d, want %d", c.month, got, c.wantRate)
			}
		})
	}
}
