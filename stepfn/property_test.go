package stepfn

import (
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
)

func genWritten() *rapid.Generator[[]relation.Row[int]] {
	return rapid.Custom(func(t *rapid.T) []relation.Row[int] {
		years := rapid.SliceOfNDistinct(
			rapid.IntRange(2018, 2090),
			1, 12,
			func(y int) int { return y },
		).Draw(t, "years")
		slices.Sort(years)

		rows := make([]relation.Row[int], 0, len(years))
		for _, y := range years {
			rows = append(rows, relation.Row[int]{
				Year:  date.Year(y),
				Value: rapid.IntRange(-1_000, 1_000).Draw(t, "value"),
			})
		}
		return rows
	})
}

func expandDrawn(t *rapid.T) ([]relation.Row[int], date.Year, date.Year, relation.Table[int]) {
	written := genWritten().Draw(t, "written")
	first := int(written[0].Year)
	from := date.Year(rapid.IntRange(first, first+40).Draw(t, "from"))
	to := from + date.Year(rapid.IntRange(0, 40).Draw(t, "length"))

	got, err := Expand(written, from, to)
	if err != nil {
		t.Fatalf("Expand(%v, %d, %d): %v", written, from, to, err)
	}
	return written, from, to, got
}

func TestExpandShouldCoverExactlyTheSpan(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		_, from, to, got := expandDrawn(t)

		want := int(to-from) + 1
		if got.Len() != want {
			t.Fatalf("the span %d..%d has %d years but the table has %d", from, to, want, got.Len())
		}
		for y := from; y <= to; y++ {
			if _, ok := got.At(y); !ok {
				t.Fatalf("year %d is missing from the expansion", y)
			}
		}
	})
}

func TestExpandShouldKeepTheWrittenValueAtItsOwnYear(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		written, from, to, got := expandDrawn(t)

		for _, row := range written {
			if row.Year < from || row.Year > to {
				continue
			}
			value, ok := got.At(row.Year)
			if !ok {
				t.Fatalf("year %d is missing from the expansion", row.Year)
			}
			if value != row.Value {
				t.Fatalf("year %d was written as %d but expanded to %d", row.Year, row.Value, value)
			}
		}
	})
}

func TestExpandShouldOnlyChangeValueAtAWrittenYear(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		written, _, _, got := expandDrawn(t)

		writtenYears := make([]date.Year, 0, len(written))
		for _, row := range written {
			writtenYears = append(writtenYears, row.Year)
		}

		rows := got.Rows()
		for i := 1; i < len(rows); i++ {
			if rows[i].Value == rows[i-1].Value {
				continue
			}
			if !slices.Contains(writtenYears, rows[i].Year) {
				t.Fatalf("the value changed at year %d, which was never written", rows[i].Year)
			}
		}
	})
}

func TestExpandShouldBeIdempotentOnAnAlreadyExpandedTable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		_, from, to, once := expandDrawn(t)

		twice, err := Expand(once.Rows(), from, to)

		if err != nil {
			t.Fatalf("expanding an expanded table failed: %v", err)
		}
		if !slices.Equal(once.Years(), twice.Years()) {
			t.Fatalf("the second expansion changed the years")
		}
		for _, row := range once.Rows() {
			second, _ := twice.At(row.Year)
			if second != row.Value {
				t.Fatalf("year %d changed from %d to %d on the second expansion", row.Year, row.Value, second)
			}
		}
	})
}
