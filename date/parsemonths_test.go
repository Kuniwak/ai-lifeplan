package date

import "testing"

func TestParseMonthsShouldReadTheMonthsAsWritten(t *testing.T) {
	for _, c := range []struct {
		written string
		want    Months
	}{
		{"", NoMonths},
		{"3", MonthOnly(3)},
		{"2,3", MonthOnly(2).Union(MonthOnly(3))},
		{"12,1", MonthOnly(12).Union(MonthOnly(1))},
		{" 2 , 3 ", MonthOnly(2).Union(MonthOnly(3))},
	} {
		t.Run(c.written, func(t *testing.T) {
			got, err := ParseMonths(c.written)
			if err != nil {
				t.Fatalf("ParseMonths(%q): %v", c.written, err)
			}
			if got != c.want {
				t.Errorf("ParseMonths(%q) = %v, want %v", c.written, got, c.want)
			}
		})
	}
}

func TestParseMonthsShouldRefuseWhatIsNotAMonth(t *testing.T) {
	for _, written := range []string{"0", "13", "-1", "2,2", "三月", "2,", ",", "2..3"} {
		t.Run(written, func(t *testing.T) {
			if got, err := ParseMonths(written); err == nil {
				t.Errorf("ParseMonths(%q) = %v, want an error", written, got)
			}
		})
	}
}
