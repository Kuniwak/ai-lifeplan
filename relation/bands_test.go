package relation

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"
)

func incomeTaxBands() Bands[int, string] {
	return NewBands([]Band[int, string]{
		{Lower: 0, Value: "5%"},
		{Lower: 1_950_000, Value: "10%"},
		{Lower: 3_300_000, Value: "20%"},
		{Lower: 6_950_000, Value: "23%"},
	})
}

func TestBandsLookup(t *testing.T) {
	type testCase struct {
		Key      int
		Expected string
	}

	testCases := map[string]testCase{
		"the very bottom (lower boundary value)":   {Key: 0, Expected: "5%"},
		"inside the first band (representative)":   {Key: 1_000_000, Expected: "5%"},
		"one below a boundary (boundary value)":    {Key: 1_949_999, Expected: "5%"},
		"exactly on a boundary (boundary value)":   {Key: 1_950_000, Expected: "10%"},
		"one above a boundary (boundary value)":    {Key: 1_950_001, Expected: "10%"},
		"inside a middle band (representative)":    {Key: 5_000_000, Expected: "20%"},
		"exactly on the last boundary (boundary)":  {Key: 6_950_000, Expected: "23%"},
		"far above the last boundary (open ended)": {Key: 900_000_000, Expected: "23%"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			bands := incomeTaxBands()

			got := bands.Lookup(tc.Key)

			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("Lookup(%d) mismatch (-want +got):\n%s", tc.Key, diff)
			}
		})
	}
}

func TestBandsLookupShouldPanicBelowTheFirstBand(t *testing.T) {
	bands := incomeTaxBands()

	msg, refused := panictest.Message(func() { bands.Lookup(-1) })

	if !refused {
		t.Fatal("want panic for a key below every band, got none")
	}
	if !strings.Contains(msg, "-1") {
		t.Errorf("the panic does not name the key: %q", msg)
	}
}

func TestBandsLookupShouldPanicWhenThereAreNoBands(t *testing.T) {
	var bands Bands[int, string]

	_, refused := panictest.Message(func() { bands.Lookup(0) })

	if !refused {
		t.Fatal("want panic when the table has no bands, got none")
	}
}

func TestNewBandsShouldOrderByLowerBound(t *testing.T) {
	bands := NewBands([]Band[int, string]{
		{Lower: 3_300_000, Value: "20%"},
		{Lower: 0, Value: "5%"},
		{Lower: 1_950_000, Value: "10%"},
	})

	if got := bands.Lookup(2_000_000); got != "10%" {
		t.Errorf("Lookup(2000000) = %q, want \"10%%\"", got)
	}
	if got, ok := bands.Min(); !ok || got != 0 {
		t.Errorf("Min() = (%d, %v), want (0, true)", got, ok)
	}
	if got := bands.Len(); got != 3 {
		t.Errorf("Len() = %d, want 3", got)
	}
}

func TestNewBandsShouldPanicOnADuplicatedLowerBound(t *testing.T) {
	refused := panictest.Recovered(func() {
		NewBands([]Band[int, string]{
			{Lower: 0, Value: "5%"},
			{Lower: 0, Value: "10%"},
		})
	})

	if refused == nil {
		t.Error("want panic for two bands starting at the same key, got none")
	}
}

func TestBandsMinOnAnEmptyTable(t *testing.T) {
	var bands Bands[int, string]

	if _, ok := bands.Min(); ok {
		t.Error("Min on a table with no bands reported a value")
	}
	if got := bands.Len(); got != 0 {
		t.Errorf("Len() = %d, want 0", got)
	}
}

func TestBandsMax(t *testing.T) {
	last, ok := incomeTaxBands().Max()
	if !ok {
		t.Fatal("帯があるのに Max が無いと言っている")
	}
	if want := 6_950_000; last != want {
		t.Errorf("最後の帯の下限が %d である（%d のはず）", last, want)
	}
}

func TestBandsMaxOnAnEmptyTable(t *testing.T) {
	if _, ok := (Bands[int, string]{}).Max(); ok {
		t.Error("帯が無いのに Max があると言っている")
	}
}
