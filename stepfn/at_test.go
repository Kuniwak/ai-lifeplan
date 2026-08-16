package stepfn_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/stepfn"
)

func written() []relation.Row[int] {
	return []relation.Row[int]{
		{Year: 2031, Value: 1_000_000},
		{Year: 2053, Value: 0},
	}
}

func TestAtShouldTakeTheLatestRowAtOrBeforeTheYear(t *testing.T) {
	cases := map[string]struct {
		at   date.Year
		want int
	}{
		"the year of a row":    {at: 2031, want: 1_000_000},
		"a year with no row":   {at: 2052, want: 1_000_000},
		"the year of the next": {at: 2053, want: 0},
		"after the last row":   {at: 2090, want: 0},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := stepfn.At(written(), c.at)
			if err != nil {
				t.Fatalf("stepfn.At: %v", err)
			}
			if got != c.want {
				t.Errorf("stepfn.At(%d) = %d, want %d", c.at, got, c.want)
			}
		})
	}
}

func TestAtShouldNotNeedTheRowsInOrder(t *testing.T) {
	got, err := stepfn.At([]relation.Row[int]{
		{Year: 2053, Value: 0},
		{Year: 2031, Value: 1_000_000},
	}, 2052)
	if err != nil {
		t.Fatalf("stepfn.At: %v", err)
	}
	if got != 1_000_000 {
		t.Errorf("stepfn.At(2052) = %d, want 1000000", got)
	}
}

func TestAtShouldRefuseATableItCannotAnswerFrom(t *testing.T) {
	cases := map[string]struct {
		written  []relation.Row[int]
		at       date.Year
		mentions string
	}{
		"a year written twice": {
			written: []relation.Row[int]{
				{Year: 2031, Value: 1_000_000},
				{Year: 2031, Value: 2_000_000},
			},
			at:       2052,
			mentions: "2031",
		},
		"a year before the first row": {
			written:  written(),
			at:       2030,
			mentions: "2031",
		},
		"nothing written at all": {
			written:  []relation.Row[int]{},
			at:       2052,
			mentions: "2052",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := stepfn.At(c.written, c.at)
			if err == nil {
				t.Fatalf("stepfn.At: 答えの無い表を受け入れた")
			}
			if !strings.Contains(err.Error(), c.mentions) {
				t.Errorf("stepfn.At: %q に触れていない: %v", c.mentions, err)
			}
		})
	}
}
