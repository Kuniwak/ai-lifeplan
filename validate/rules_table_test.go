package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func check(t *testing.T, rule Rule, slot tsv.Slot, table *tsv.Table) []string {
	t.Helper()

	return messagesOf(Run([]Rule{rule}, map[tsv.Slot]*tsv.Table{slot: table}, RequireAll).Findings)
}

func TestColumnSchema(t *testing.T) {
	columns := []Column{
		{Name: "西暦", Parse: AsYear},
		{Name: "年収", Parse: AsYen},
		{Name: "料率", Parse: AsPercent},
		{Name: "備考", Parse: AsOptional(AsText)},
	}

	type testCase struct {
		Table *tsv.Table
		Wants [][]string
	}

	testCases := map[string]testCase{
		"every column present and readable": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率", "備考"},
				Rows:   [][]string{{"2031", "1,000,000", "2.025%", ""}},
			},
		},
		"a missing column": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率"},
				Rows:   [][]string{{"2031", "1000000", "8%"}},
			},
			Wants: [][]string{{"備考"}},
		},
		"an amount that is not one": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率", "備考"},
				Rows:   [][]string{{"2031", "120万", "8%", ""}},
			},
			Wants: [][]string{{"row 1", "年収"}},
		},
		"a rate without its percent sign": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率", "備考"},
				Rows:   [][]string{{"2031", "1000000", "8", ""}},
			},
			Wants: [][]string{{"料率"}},
		},
		"a decimal amount": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率", "備考"},
				Rows:   [][]string{{"2031", "1000.5", "8%", ""}},
			},
			Wants: [][]string{{"年収"}},
		},
		"a blank in a column that does not allow it": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率", "備考"},
				Rows:   [][]string{{"2031", "", "8%", ""}},
			},
			Wants: [][]string{{"年収"}},
		},
		"every offending field is reported": {
			Table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "年収", "料率", "備考"},
				Rows:   [][]string{{"2031", "x", "8%", ""}, {"2032", "y", "8%", ""}},
			},
			Wants: [][]string{{"row 1", "年収"}, {"row 2", "年収"}},
		},
	}

	for name, tc := range testCases {
		t.Run(string(name), func(t *testing.T) {

			got := check(t, ColumnSchema("income", columns), "income", tc.Table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestYearGap(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Wants [][]string
	}

	testCases := map[string]testCase{
		"every year of the span": {
			Rows: [][]string{{"2031"}, {"2032"}, {"2033"}},
		},
		"a year missing in the middle": {
			Rows:  [][]string{{"2031"}, {"2033"}},
			Wants: [][]string{{"2032", "missing"}},
		},
		"a year missing at the end (boundary value)": {
			Rows:  [][]string{{"2031"}, {"2032"}},
			Wants: [][]string{{"2033", "missing"}},
		},
		"a year missing at the start (boundary value)": {
			Rows:  [][]string{{"2032"}, {"2033"}},
			Wants: [][]string{{"2031", "missing"}},
		},
		"a year written twice": {
			Rows:  [][]string{{"2031"}, {"2031"}, {"2032"}, {"2033"}},
			Wants: [][]string{{"2031", "二度書かれており"}},
		},
		"a year outside the span": {
			Rows:  [][]string{{"2031"}, {"2032"}, {"2033"}, {"2099"}},
			Wants: [][]string{{"2099", "outside"}},
		},
		"nothing at all": {
			Rows:  nil,
			Wants: [][]string{{"2031", "missing"}, {"2032", "missing"}, {"2033", "missing"}},
		},
	}

	for name, tc := range testCases {
		t.Run(string(name), func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"西暦"}, Rows: tc.Rows}

			got := check(t, YearGap("calendar", "西暦", 2031, 2033), "calendar", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestStepMonotonic(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Wants [][]string
	}

	testCases := map[string]testCase{
		"ascending years with gaps are fine": {
			Rows: [][]string{{"2031"}, {"2053"}, {"2058"}},
		},
		"a single row": {
			Rows: [][]string{{"2031"}},
		},
		"no rows": {
			Rows: nil,
		},
		"the years run backwards": {
			Rows:  [][]string{{"2053"}, {"2031"}},
			Wants: [][]string{{"2031", "昇順でなければならない"}},
		},
		"the same year twice": {
			Rows:  [][]string{{"2031"}, {"2031"}},
			Wants: [][]string{{"2031", "二度書かれており"}},
		},
		"a year that is not a number": {
			Rows:  [][]string{{"令和3年"}},
			Wants: [][]string{{"令和3年"}},
		},
	}

	for name, tc := range testCases {
		t.Run(string(name), func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"西暦"}, Rows: tc.Rows}

			got := check(t, StepMonotonic("income", "西暦"), "income", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestStepCoversStart(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Wants [][]string
	}

	testCases := map[string]testCase{
		"the first row is the year the plan starts (boundary value)": {
			Rows: [][]string{{"2018"}, {"2031"}},
		},
		"the first row is before the plan starts": {
			Rows: [][]string{{"1989"}, {"2031"}},
		},
		"the first row is after the plan starts": {
			Rows:  [][]string{{"2019"}, {"2031"}},
			Wants: [][]string{{"2019", "2018"}},
		},
		"nothing is written": {
			Rows:  nil,
			Wants: [][]string{{"2018"}},
		},
	}

	for name, tc := range testCases {
		t.Run(string(name), func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"西暦"}, Rows: tc.Rows}

			got := check(t, StepCoversStart("income", "西暦", 2018), "income", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestValueRange(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Wants [][]string
	}

	testCases := map[string]testCase{
		"inside the range": {
			Rows: [][]string{{"0"}, {"1,000,000"}},
		},
		"exactly on the bounds (boundary value)": {
			Rows: [][]string{{"0"}, {"9999999"}},
		},
		"below the lower bound": {
			Rows:  [][]string{{"-1"}},
			Wants: [][]string{{"-1", "outside"}},
		},
		"above the upper bound": {
			Rows:  [][]string{{"10000000"}},
			Wants: [][]string{{"10000000", "outside"}},
		},
		"an unreadable value is left to column-schema": {
			Rows: [][]string{{"x"}},
		},
	}

	for name, tc := range testCases {
		t.Run(string(name), func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"保険料"}, Rows: tc.Rows}

			got := check(t, ValueRange("insurance", "保険料", 0, 9_999_999), "insurance", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestRulesShouldReportAMissingColumnRatherThanCrash(t *testing.T) {
	table := &tsv.Table{Header: []tsv.ColumnName{"別の列"}, Rows: [][]string{{"x"}}}
	rules := map[RuleName]Rule{
		YearGapRule:         YearGap("s", "西暦", 2031, 2033),
		StepMonotonicRule:   StepMonotonic("s", "西暦"),
		StepCoversStartRule: StepCoversStart("s", "西暦", 2031),
		ValueRangeRule:      ValueRange("s", "金額", 0, 1),
	}

	for name, rule := range rules {
		t.Run(string(name), func(t *testing.T) {
			got := check(t, rule, "s", table)

			assertFindings(t, got, []string{"missing"})
		})
	}
}
