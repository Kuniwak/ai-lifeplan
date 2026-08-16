package money

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseYenOK(t *testing.T) {
	type testCase struct {
		Input    string
		Expected Yen
	}

	testCases := map[string]testCase{
		"zero (boundary value)":             {Input: "0", Expected: 0},
		"plain integer (representative)":    {Input: "12000000", Expected: 12000000},
		"comma separated (representative)":  {Input: "1,000,000", Expected: 1000000},
		"underscore separated (represent.)": {Input: "1_200_000", Expected: 1200000},
		"mixed separators (representative)": {Input: "1,200_000", Expected: 1200000},
		"irregular separator position":      {Input: "1,2,3", Expected: 123},
		"negative (representative)":         {Input: "-580,000", Expected: -580000},
		"explicit plus sign":                {Input: "+1,000", Expected: 1000},
		"surrounding spaces are ignored":    {Input: "  1,000  ", Expected: 1000},
		"single digit (lower boundary)":     {Input: "1", Expected: 1},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got, err := ParseYen(tc.Input)

			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("ParseYen(%q) mismatch (-want +got):\n%s", tc.Input, diff)
			}
		})
	}
}

func TestParseYenNG(t *testing.T) {
	testCases := map[string]string{
		"empty is not zero (unrecorded differs from 0)": "",
		"blank only":                     "   ",
		"decimal point is rejected":      "1000.5",
		"decimal zero is still rejected": "1000.0",
		"man-yen notation is rejected":   "120万",
		"unit suffix is rejected":        "1000円",
		"separator only":                 ",",
		"sign only":                      "-",
		"not a number":                   "abc",
		"embedded space":                 "1 000",
	}

	for name, input := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := ParseYen(input)

			if err == nil {
				t.Errorf("ParseYen(%q): want error, got none", input)
			}
		})
	}
}

func TestYenStringShouldNotContainSeparators(t *testing.T) {
	type testCase struct {
		Input    Yen
		Expected string
	}

	testCases := map[string]testCase{
		"zero (boundary value)":     {Input: 0, Expected: "0"},
		"large (representative)":    {Input: 12000000, Expected: "12000000"},
		"negative (representative)": {Input: -580000, Expected: "-580000"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := tc.Input.String()

			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("String() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
