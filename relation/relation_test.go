package relation

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"
)

func TestNewShouldOrderRowsByYear(t *testing.T) {
	rows := []Row[int]{
		{Year: 2033, Value: 3},
		{Year: 2031, Value: 1},
		{Year: 2032, Value: 2},
	}

	got := New(rows)

	wantYears := []date.Year{2031, 2032, 2033}
	if diff := cmp.Diff(wantYears, got.Years()); diff != "" {
		t.Errorf("Years() mismatch (-want +got):\n%s", diff)
	}

	wantRows := []Row[int]{{Year: 2031, Value: 1}, {Year: 2032, Value: 2}, {Year: 2033, Value: 3}}
	if diff := cmp.Diff(wantRows, got.Rows()); diff != "" {
		t.Errorf("Rows() mismatch (-want +got):\n%s", diff)
	}
}

func TestNewShouldNotShareStorageWithTheCaller(t *testing.T) {
	rows := []Row[int]{{Year: 2031, Value: 1}, {Year: 2032, Value: 2}}
	table := New(rows)

	rows[0].Value = 999

	got, ok := table.At(2031)
	if !ok {
		t.Fatal("At(2031): want a row, got none")
	}
	if got != 1 {
		t.Errorf("the caller's slice reached into the table: want 1, got %d", got)
	}
}

func TestNewShouldPanicOnADuplicatedYear(t *testing.T) {
	rows := []Row[int]{{Year: 2031, Value: 1}, {Year: 2031, Value: 2}}

	refused := panictest.Recovered(func() { New(rows) })

	if refused == nil {
		t.Error("want panic for a duplicated year, got none")
	}
}

func TestConstantShouldHoldTheValueInEveryOneOfTheYears(t *testing.T) {
	years := []date.Year{2033, 2031, 2032}

	got := Constant(years, 7)

	if want := []date.Year{2031, 2032, 2033}; !cmp.Equal(got.Years(), want) {
		t.Errorf("years = %v, want %v", got.Years(), want)
	}
	for _, y := range years {
		v, ok := got.At(y)
		if !ok {
			t.Errorf("year %d is missing", y)
			continue
		}
		if v != 7 {
			t.Errorf("year %d holds %d, want 7", y, v)
		}
	}
	if got.Len() != len(years) {
		t.Errorf("Len = %d, want %d", got.Len(), len(years))
	}
}

func TestConstantShouldPanicOnADuplicatedYear(t *testing.T) {
	refused := panictest.Recovered(func() { Constant([]date.Year{2031, 2031, 2032}, 7) })

	if refused == nil {
		t.Error("want panic for a duplicated year, got none")
	}
}

func TestConstantShouldReturnAnEmptyTableForNoYears(t *testing.T) {
	got := Constant(nil, 7)

	if got.Len() != 0 {
		t.Errorf("Len = %d, want 0", got.Len())
	}
}

func TestSpanShouldRunFromTheFirstYearToTheLast(t *testing.T) {
	for name, c := range map[string]struct {
		from, to date.Year
		want     []date.Year
	}{
		"複数年":         {2031, 2033, []date.Year{2031, 2032, 2033}},
		"1 年（境界値）":    {2031, 2031, []date.Year{2031}},
		"逆向き（境界値）":    {2032, 2031, []date.Year{}},
		"ずっと逆向き（境界値）": {2033, 2031, []date.Year{}},
	} {
		t.Run(name, func(t *testing.T) {
			got := Span(c.from, c.to)

			if !cmp.Equal(got, c.want) {
				t.Errorf("Span(%d, %d) = %v, want %v", c.from, c.to, got, c.want)
			}
		})
	}
}

func TestTableAccessors(t *testing.T) {
	table := New([]Row[string]{{Year: 2031, Value: "a"}, {Year: 2032, Value: "b"}})

	if got := table.Len(); got != 2 {
		t.Errorf("Len() = %d, want 2", got)
	}
	if got, ok := table.At(2032); !ok || got != "b" {
		t.Errorf(`At(2032) = (%q, %v), want ("b", true)`, got, ok)
	}
	if _, ok := table.At(2099); ok {
		t.Error("At(2099) reported a row that was never added")
	}
}

func TestEmptyTable(t *testing.T) {
	var table Table[int]

	if got := table.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
	if got := table.Years(); got != nil {
		t.Errorf("Years() = %v, want nil", got)
	}
	if _, ok := table.At(2031); ok {
		t.Error("At on an empty table reported a row")
	}
}

func TestOverShouldLayAValueOnEveryYearItIsGiven(t *testing.T) {
	got := Over(Span(2020, 2023), func(y date.Year) int { return int(y) * 2 })

	if got.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", got.Len())
	}
	for _, y := range []date.Year{2020, 2021, 2022, 2023} {
		v, ok := got.At(y)
		if !ok {
			t.Errorf("%d が無い", y)
			continue
		}
		if want := int(y) * 2; v != want {
			t.Errorf("At(%d) = %d, want %d", y, v, want)
		}
	}
}

func TestOverShouldAnswerABackwardsSpanTheWayConstantDoes(t *testing.T) {
	backwards := Span(2023, 2020)
	if len(backwards) != 0 {
		t.Fatalf("Span(2023, 2020) = %v, want no years", backwards)
	}
	if got := Over(backwards, func(date.Year) int { return 1 }); got.Len() != 0 {
		t.Errorf("Over(逆向きの span) の表に %d 行ある。空のはず", got.Len())
	}
}

func TestOverShouldWalkTheYearsAscendingWhateverOrderTheyCameIn(t *testing.T) {
	var called []date.Year
	got := Over([]date.Year{2023, 2020, 2022}, func(y date.Year) int {
		called = append(called, y)
		return len(called)
	})

	if want := []date.Year{2020, 2022, 2023}; !slices.Equal(called, want) {
		t.Errorf("valueAt が %v の順で呼ばれた。%v のはず", called, want)
	}
	for y, want := range map[date.Year]int{2020: 1, 2022: 2, 2023: 3} {
		if v, _ := got.At(y); v != want {
			t.Errorf("At(%d) = %d, want %d", y, v, want)
		}
	}
}
