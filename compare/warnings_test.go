package compare_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestWarningsShouldReportProjectsThatStartFromDifferentYears(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2025},
		{Name: "as-of-2022", StartsAfter: 2022},
	}

	got := compare.Warnings(subjects)

	if len(got) != 1 {
		t.Fatalf("warnings = %v, want exactly 1", got)
	}
	for _, want := range []string{"base", "2025", "as-of-2022", "2022"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("%q が %q を名指ししていない", got[0], want)
		}
	}
}

func TestWarningsShouldSaySoWhenTheProjectsStartFromTheSameYear(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2025},
		{Name: "settle-2050", StartsAfter: 2025},
	}

	if got := compare.Warnings(subjects); len(got) != 0 {
		t.Errorf("warnings = %v, want none", got)
	}
}

func TestWarningsShouldReportASlotItCannotClassify(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2025, Paths: map[tsv.Slot]string{
			"living_cost": "elsewhere/living-cost.tsv",
		}},
		{Name: "other", StartsAfter: 2025, Paths: map[tsv.Slot]string{
			"living_cost": "elsewhere/living-cost-high.tsv",
		}},
	}

	got := compare.Warnings(subjects)

	if len(got) != 1 {
		t.Fatalf("warnings = %v, want exactly 1", got)
	}
	if !strings.Contains(got[0], "living_cost") {
		t.Errorf("%q が slot を名指ししていない", got[0])
	}
}

func TestWarningsShouldSayWhichRecordTheGapWasMeasuredAgainst(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2025, Record: record(t,
			[]string{"2024", "100", "200", "300", ""},
			[]string{"2025", "110", "210", "310", ""},
		), Paths: map[tsv.Slot]string{
			"balance": "actuals/balance.tsv",
		}},
		{Name: "as-of-2022", StartsAfter: 2025, Record: record(t,
			[]string{"2024", "100", "200", "300", ""},
		), Paths: map[tsv.Slot]string{
			"balance": "actuals/balance-2022.tsv",
		}},
	}

	got := compare.Warnings(subjects)

	if len(got) != 1 {
		t.Fatalf("warnings = %v, want exactly 1", got)
	}
	for _, want := range []string{"base", "actuals/balance.tsv", "actuals/balance-2022.tsv"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("%q が %q を名指ししていない", got[0], want)
		}
	}
}

func TestWarningsShouldSaySoWhenTheProjectsReadTheSameRecord(t *testing.T) {
	held := func() map[tsv.Slot]string { return map[tsv.Slot]string{"balance": "actuals/balance.tsv"} }
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2025, Record: record(t,
			[]string{"2025", "110", "210", "310", ""},
		), Paths: held()},
		{Name: "settle-2050", StartsAfter: 2025, Record: record(t,
			[]string{"2025", "110", "210", "310", ""},
		), Paths: map[tsv.Slot]string{"balance": "./actuals/balance.tsv"}},
	}

	if got := compare.Warnings(subjects); len(got) != 0 {
		t.Errorf("warnings = %v, want none", got)
	}
}
