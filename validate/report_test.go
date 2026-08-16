package validate

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestListTableShouldShowWhatEachRuleNeeds(t *testing.T) {
	registry := NewRegistry([]Rule{
		alwaysPasses("year-coverage", "household", "market"),
		alwaysPasses("year-gap", "household"),
	})

	got := ListTable(registry)

	want := &tsv.Table{
		Header: []tsv.ColumnName{RuleColumn, NeedsColumn},
		Rows: [][]string{
			{"year-coverage", "household market"},
			{"year-gap", "household"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ListTable mismatch (-want +got):\n%s", diff)
	}
}

func TestCoverageTableShouldNameTheSkippedRules(t *testing.T) {
	result := Result{Ran: []RuleName{"year-gap"}, Skipped: []RuleName{"year-coverage"}}

	got := CoverageTable(result)

	want := &tsv.Table{
		Header: []tsv.ColumnName{RuleColumn, StatusColumn},
		Rows: [][]string{
			{"year-gap", string(StatusRan)},
			{"year-coverage", string(StatusSkipped)},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CoverageTable mismatch (-want +got):\n%s", diff)
	}
}

func TestFindingsTableShouldCarryTheRuleAndSlot(t *testing.T) {
	result := Result{Findings: []Finding{
		{Rule: "year-gap", Slot: "household", Message: "year 2045 is missing"},
	}}

	got := FindingsTable(result)

	want := &tsv.Table{
		Header: []tsv.ColumnName{RuleColumn, SlotColumn, MessageColumn},
		Rows:   [][]string{{"year-gap", "household", "year 2045 is missing"}},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FindingsTable mismatch (-want +got):\n%s", diff)
	}
}

func TestFindingStringShouldReadWithoutASlot(t *testing.T) {
	withSlot := Finding{Rule: "year-gap", Slot: "household", Message: "year 2045 is missing"}
	withoutSlot := Finding{Rule: "slot-resolvable", Message: "market is not set"}

	if got, want := withSlot.String(), "year-gap: household: year 2045 is missing"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got, want := withoutSlot.String(), "slot-resolvable: market is not set"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
