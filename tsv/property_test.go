package tsv

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

func genTable() *rapid.Generator[*Table] {
	return rapid.Custom(func(t *rapid.T) *Table {
		width := rapid.IntRange(1, 6).Draw(t, "width")

		header := make([]ColumnName, 0, width)
		for i := range width {
			name := rapid.SampledFrom([]string{"西暦", "col", "備考", "a b", "x\ty"}).Draw(t, "name")
			header = append(header, ColumnName(name+strings.Repeat("'", i)))
		}

		field := rapid.SampledFrom([]string{"", "0", "1000000", "a\tb", "a\nb", `q " q`, "西暦", " ", "-1"})

		height := rapid.IntRange(0, 8).Draw(t, "height")
		var rows [][]string
		for range height {
			row := make([]string, 0, width)
			for range width {
				row = append(row, field.Draw(t, "field"))
			}
			rows = append(rows, row)
		}

		return &Table{Header: header, Rows: rows}
	})
}

func TestWriteThenReadShouldPreserveTheTable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := genTable().Draw(t, "table")
		var sb strings.Builder
		if err := Write(&sb, original); err != nil {
			t.Fatalf("Write: %v", err)
		}

		got, err := Read(strings.NewReader(sb.String()))

		if err != nil {
			t.Fatalf("Read(%q): %v", sb.String(), err)
		}
		if diff := cmp.Diff(original, got); diff != "" {
			t.Fatalf("round trip changed the table (-want +got):\n%s", diff)
		}
	})
}

func TestWriteShouldBeDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		table := genTable().Draw(t, "table")

		var first, second strings.Builder
		if err := Write(&first, table); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if err := Write(&second, table); err != nil {
			t.Fatalf("Write: %v", err)
		}

		if first.String() != second.String() {
			t.Fatalf("two writes of the same table differ:\nfirst  %q\nsecond %q", first.String(), second.String())
		}
	})
}

func TestWriteShouldEndEveryLineWithANewline(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		table := genTable().Draw(t, "table")

		var sb strings.Builder
		if err := Write(&sb, table); err != nil {
			t.Fatalf("Write: %v", err)
		}

		if !strings.HasSuffix(sb.String(), "\n") {
			t.Fatalf("the output does not end with a newline: %q", sb.String())
		}
	})
}
