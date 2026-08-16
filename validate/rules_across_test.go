package validate

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/tsv"
)

func yearTable(years ...int) *tsv.Table {
	table := &tsv.Table{Header: []tsv.ColumnName{"西暦"}}
	for _, y := range years {
		table.Rows = append(table.Rows, []string{strconv.Itoa(y)})
	}
	return table
}

func runAcross(t *testing.T, rule Rule, present map[tsv.Slot]*tsv.Table) []string {
	t.Helper()

	return messagesWithSlotOf(Run([]Rule{rule}, present, RequireAll).Findings)
}

func TestYearCoverage(t *testing.T) {
	type testCase struct {
		Present map[tsv.Slot]*tsv.Table
		Wants   [][]string
	}

	testCases := map[string]testCase{
		"every table covers the whole span": {
			Present: map[tsv.Slot]*tsv.Table{
				"income":  yearTable(2031, 2032, 2033),
				"expense": yearTable(2031, 2032, 2033),
			},
		},
		"the order in the file does not matter": {
			Present: map[tsv.Slot]*tsv.Table{
				"income":  yearTable(2033, 2031, 2032),
				"expense": yearTable(2031, 2032, 2033),
			},
		},
		"one table is a year short": {
			Present: map[tsv.Slot]*tsv.Table{
				"income":  yearTable(2031, 2032, 2033),
				"expense": yearTable(2031, 2032),
			},
			Wants: [][]string{{"expense", "2033"}},
		},
		"one table reaches past the span": {
			Present: map[tsv.Slot]*tsv.Table{
				"income":  yearTable(2031, 2032, 2033, 2099),
				"expense": yearTable(2031, 2032, 2033),
			},
			Wants: [][]string{{"income", "2099"}},
		},
		"a table with no years at all": {
			Present: map[tsv.Slot]*tsv.Table{
				"income":  yearTable(),
				"expense": yearTable(2031, 2032, 2033),
			},
			Wants: [][]string{{"income", "2031"}},
		},
		"a year that is not a number": {
			Present: map[tsv.Slot]*tsv.Table{
				"income":  {Header: []tsv.ColumnName{"西暦"}, Rows: [][]string{{"令和3年"}}},
				"expense": yearTable(2031, 2032, 2033),
			},
			Wants: [][]string{{"income", "令和3年"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			rule := YearCoverage([]tsv.Slot{"income", "expense"}, "西暦", 2031, 2033)

			got := runAcross(t, rule, tc.Present)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestYearCoverageShouldNeedEveryTableAtOnce(t *testing.T) {
	rule := YearCoverage([]tsv.Slot{"income", "expense"}, "西暦", 2031, 2033)
	present := map[tsv.Slot]*tsv.Table{"income": yearTable(2031, 2032, 2033)}

	got := Run([]Rule{rule}, present, AllowMissing)

	if diff := cmp.Diff([]RuleName{YearCoverageRule}, got.Skipped); diff != "" {
		t.Errorf("Skipped mismatch (-want +got):\n%s", diff)
	}
	if !got.Partial() {
		t.Error("Partial() reported a complete check although the rule was skipped")
	}
}

func TestSlotResolvable(t *testing.T) {
	type testCase struct {
		Required []tsv.Slot
		Paths    map[tsv.Slot]string
		Existing []string
		Wants    [][]string
	}

	testCases := map[string]testCase{
		"every slot is set and its file is there": {
			Required: []tsv.Slot{"household", "market"},
			Paths:    map[tsv.Slot]string{"household": "data/household.tsv", "market": "data/market.tsv"},
			Existing: []string{"data/household.tsv", "data/market.tsv"},
		},
		"a slot nobody set": {
			Required: []tsv.Slot{"household", "market"},
			Paths:    map[tsv.Slot]string{"household": "data/household.tsv"},
			Existing: []string{"data/household.tsv"},
			Wants:    [][]string{{"market", "no layer sets"}},
		},
		"a slot pointing at a file that is not there": {
			Required: []tsv.Slot{"household"},
			Paths:    map[tsv.Slot]string{"household": "data/typo.tsv"},
			Existing: []string{"data/household.tsv"},
			Wants:    [][]string{{"household", "data/typo.tsv"}},
		},
		"a slot the plan does not require is not complained about": {
			Required: []tsv.Slot{"household"},
			Paths:    map[tsv.Slot]string{"household": "data/household.tsv", "spare": "nowhere.tsv"},
			Existing: []string{"data/household.tsv"},
		},
		"nothing is required": {
			Required: nil,
			Paths:    map[tsv.Slot]string{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			existing := make(map[string]bool, len(tc.Existing))
			for _, p := range tc.Existing {
				existing[p] = true
			}
			rule := SlotResolvable(tc.Required, tc.Paths, func(p string) bool { return existing[p] })

			got := runAcross(t, rule, map[tsv.Slot]*tsv.Table{})

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestSlotResolvableShouldRunEvenWithNoTables(t *testing.T) {
	rule := SlotResolvable([]tsv.Slot{"household"}, map[tsv.Slot]string{}, func(string) bool { return false })

	got := Run([]Rule{rule}, map[tsv.Slot]*tsv.Table{}, AllowMissing)

	if diff := cmp.Diff([]RuleName{SlotResolvableRule}, got.Ran); diff != "" {
		t.Errorf("the rule did not run (-want +got):\n%s", diff)
	}
	if got.OK() {
		t.Error("OK() reported success although a required slot is unset")
	}
}
