package input_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestUnreadColumnsShouldNameWhatNoShapeDeclares(t *testing.T) {
	table := &tsv.Table{
		Header: []tsv.ColumnName{input.YearColumn, input.InflationRateColumn, "メモ", "インフレ立"},
		Rows:   [][]string{{"2018", "0.00%", "総務省", "x"}},
	}

	got := input.UnreadColumns(input.InflationSlot, table)

	if diff := cmp.Diff([]tsv.ColumnName{"インフレ立", "メモ"}, got); diff != "" {
		t.Errorf("UnreadColumns mismatch (-want +got):\n%s", diff)
	}
}

func TestUnreadColumnsShouldSayNothingAboutATableWrittenAsDeclared(t *testing.T) {
	table := &tsv.Table{
		Header: []tsv.ColumnName{input.YearColumn, input.InflationRateColumn},
		Rows:   [][]string{{"2018", "0.00%"}},
	}

	got := input.UnreadColumns(input.InflationSlot, table)

	if len(got) != 0 {
		t.Errorf("UnreadColumns = %v, want none", got)
	}
}

func TestUnreadColumnsShouldSayNothingAboutASlotNoShapeDescribes(t *testing.T) {
	table := &tsv.Table{Header: []tsv.ColumnName{"何か"}, Rows: nil}

	got := input.UnreadColumns("no-such-slot", table)

	if len(got) != 0 {
		t.Errorf("UnreadColumns = %v, want none (nothing is known about the slot)", got)
	}
}
