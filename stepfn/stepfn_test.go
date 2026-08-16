package stepfn

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
)

func TestExpandOK(t *testing.T) {
	type testCase struct {
		Written  []relation.Row[int]
		From     date.Year
		To       date.Year
		Expected []relation.Row[int]
	}

	testCases := map[string]testCase{
		"a value holds until the next written year (representative)": {
			Written: []relation.Row[int]{{Year: 2031, Value: 100}, {Year: 2033, Value: 0}},
			From:    2031, To: 2034,
			Expected: []relation.Row[int]{
				{Year: 2031, Value: 100},
				{Year: 2032, Value: 100},
				{Year: 2033, Value: 0},
				{Year: 2034, Value: 0},
			},
		},
		"the last written value runs to the end (open ended)": {
			Written: []relation.Row[int]{{Year: 2031, Value: 7}},
			From:    2031, To: 2033,
			Expected: []relation.Row[int]{
				{Year: 2031, Value: 7},
				{Year: 2032, Value: 7},
				{Year: 2033, Value: 7},
			},
		},
		"a single year span (boundary value)": {
			Written: []relation.Row[int]{{Year: 2031, Value: 7}},
			From:    2031, To: 2031,
			Expected: []relation.Row[int]{{Year: 2031, Value: 7}},
		},
		"writing every year changes nothing": {
			Written: []relation.Row[int]{{Year: 2031, Value: 1}, {Year: 2032, Value: 2}},
			From:    2031, To: 2032,
			Expected: []relation.Row[int]{{Year: 2031, Value: 1}, {Year: 2032, Value: 2}},
		},
		"the span may start after the first written year": {
			Written: []relation.Row[int]{{Year: 2020, Value: 5}, {Year: 2031, Value: 9}},
			From:    2030, To: 2031,
			Expected: []relation.Row[int]{{Year: 2030, Value: 5}, {Year: 2031, Value: 9}},
		},
		"a written year after the end is not carried in": {
			Written: []relation.Row[int]{{Year: 2031, Value: 1}, {Year: 2040, Value: 2}},
			From:    2031, To: 2032,
			Expected: []relation.Row[int]{{Year: 2031, Value: 1}, {Year: 2032, Value: 1}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got, err := Expand(tc.Written, tc.From, tc.To)

			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if diff := cmp.Diff(tc.Expected, got.Rows()); diff != "" {
				t.Errorf("Expand mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandNG(t *testing.T) {
	type testCase struct {
		Written  []relation.Row[int]
		From     date.Year
		To       date.Year
		Mentions string
	}

	testCases := map[string]testCase{
		"the span starts before the first written year": {
			Written:  []relation.Row[int]{{Year: 2031, Value: 100}},
			From:     2030,
			To:       2033,
			Mentions: "2030",
		},
		"nothing was written at all": {
			Written:  nil,
			From:     2031,
			To:       2033,
			Mentions: "2031",
		},
		"the same year twice": {
			Written:  []relation.Row[int]{{Year: 2031, Value: 1}, {Year: 2031, Value: 2}},
			From:     2031,
			To:       2033,
			Mentions: "2031",
		},
		"the years run backwards": {
			Written:  []relation.Row[int]{{Year: 2033, Value: 1}, {Year: 2031, Value: 2}},
			From:     2031,
			To:       2033,
			Mentions: "2031",
		},
		"the span runs backwards": {
			Written:  []relation.Row[int]{{Year: 2031, Value: 1}},
			From:     2033,
			To:       2031,
			Mentions: "2033",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := Expand(tc.Written, tc.From, tc.To)

			if err == nil {
				t.Fatal("want error, got none")
			}
			if !strings.Contains(err.Error(), tc.Mentions) {
				t.Errorf("the message does not mention %q: %v", tc.Mentions, err)
			}
		})
	}
}
