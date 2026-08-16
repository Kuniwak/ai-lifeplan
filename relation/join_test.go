package relation

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"
)

func TestJoinShouldCombineMatchingYears(t *testing.T) {
	income := New([]Row[int]{{2031, 1000}, {2032, 1100}, {2033, 1200}})
	expense := New([]Row[int]{{2031, 600}, {2032, 700}, {2033, 800}})

	got := Join(income, expense, func(y date.Year, in, ex int) int { return in - ex })

	want := []Row[int]{{2031, 400}, {2032, 400}, {2033, 400}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Join mismatch (-want +got):\n%s", diff)
	}
}

func TestJoinShouldPassTheYearToTheCombiner(t *testing.T) {
	a := New([]Row[int]{{2031, 1}})
	b := New([]Row[int]{{2031, 2}})

	got := Join(a, b, func(y date.Year, av, bv int) date.Year { return y })

	want := []Row[date.Year]{{2031, 2031}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Join mismatch (-want +got):\n%s", diff)
	}
}

func TestJoinShouldAcceptTwoEmptyTables(t *testing.T) {
	var a, b Table[int]

	got := Join(a, b, func(y date.Year, av, bv int) int { return av + bv })

	if got.Len() != 0 {
		t.Errorf("Len() = %d, want 0", got.Len())
	}
}

func TestJoinShouldPanicWhenTheYearSetsDiffer(t *testing.T) {
	type testCase struct {
		A            Table[int]
		B            Table[int]
		WantMentions []string
	}

	testCases := map[string]testCase{
		"the left table covers a year the right does not": {
			A:            New([]Row[int]{{2031, 1}, {2032, 1}, {2090, 1}}),
			B:            New([]Row[int]{{2031, 1}, {2032, 1}}),
			WantMentions: []string{"2090"},
		},
		"the right table covers a year the left does not": {
			A:            New([]Row[int]{{2031, 1}}),
			B:            New([]Row[int]{{2031, 1}, {2092, 1}}),
			WantMentions: []string{"2092"},
		},
		"both sides have a year the other lacks": {
			A:            New([]Row[int]{{2031, 1}, {2090, 1}}),
			B:            New([]Row[int]{{2031, 1}, {2092, 1}}),
			WantMentions: []string{"2092", "2090"},
		},
		"one side is empty": {
			A:            New([]Row[int]{{2031, 1}}),
			B:            Table[int]{},
			WantMentions: []string{"2031"},
		},
		"the same number of years but not the same years": {
			A:            New([]Row[int]{{2031, 1}, {2032, 1}}),
			B:            New([]Row[int]{{2031, 1}, {2033, 1}}),
			WantMentions: []string{"2032", "2033"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			msg, refused := panictest.Message(func() {
				Join(tc.A, tc.B, func(y date.Year, av, bv int) int { return av + bv })
			})

			if !refused {
				t.Fatal("want panic, got none")
			}
			for _, want := range tc.WantMentions {
				if !strings.Contains(msg, want) {
					t.Errorf("the panic does not name year %s, so the cause cannot be found: %q", want, msg)
				}
			}
		})
	}
}

func TestLeftJoinShouldUseTheGivenValueForAMissingYear(t *testing.T) {
	years := New([]Row[int]{{2031, 1}, {2032, 2}, {2033, 3}})
	extraordinary := New([]Row[int]{{2032, 300}})

	got := LeftJoin(years, extraordinary, 0, func(y date.Year, base, extra int) int { return base + extra })

	want := []Row[int]{{2031, 1}, {2032, 302}, {2033, 3}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("LeftJoin mismatch (-want +got):\n%s", diff)
	}
}

func TestLeftJoinShouldKeepTheYearsOfTheLeftTable(t *testing.T) {
	left := New([]Row[int]{{2031, 1}, {2032, 2}})
	var right Table[int]

	got := LeftJoin(left, right, -1, func(y date.Year, l, r int) int { return r })

	want := []Row[int]{{2031, -1}, {2032, -1}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("LeftJoin mismatch (-want +got):\n%s", diff)
	}
}

func TestLeftJoinShouldPanicWhenTheRightTableHasAnUnknownYear(t *testing.T) {
	left := New([]Row[int]{{2031, 1}, {2032, 2}})
	right := New([]Row[int]{{2032, 300}, {2099, 400}})

	msg, refused := panictest.Message(func() {
		LeftJoin(left, right, 0, func(y date.Year, l, r int) int { return l + r })
	})

	if !refused {
		t.Fatal("want panic for a year only the right table has, got none")
	}
	if !strings.Contains(msg, "2099") {
		t.Errorf("the panic does not name year 2099: %q", msg)
	}
}
