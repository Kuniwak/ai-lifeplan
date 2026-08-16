package sheets_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
)

func TestManYenShouldReadEveryShapeTheCopyWrites(t *testing.T) {
	for _, c := range []struct {
		in   string
		want money.Yen
	}{
		{"0", 0},
		{"215.3", 2_153_000},
		{"1,002.8", 10_028_000},
		{"-58.7", -587_000},
		{"0.1", 1_000},
		{"0.01", 100},
		{"0.0001", 1},
		{"-0.01", -100},
	} {
		got, err := sheets.ManYen(c.in)
		if err != nil {
			t.Errorf("ManYen(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ManYen(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestManYenShouldRefuseWhatItCannotHold(t *testing.T) {
	for _, in := range []string{
		"",
		"16.6万/月",
		"０",
		"0.00001",
	} {
		if got, err := sheets.ManYen(in); err == nil {
			t.Errorf("ManYen(%q) = %d, want an error", in, got)
		}
	}
}
