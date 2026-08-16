package compare_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestSlotsShouldNameTheSlotThatDiffersAndItsClass(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Paths: map[tsv.Slot]string{
			"income_wife": "data/controllable/income-wife.tsv",
			"inflation":   "data/environment/inflation.tsv",
		}},
		{Name: "wife-fulltime", Paths: map[tsv.Slot]string{
			"income_wife": "data/controllable/scenario/income-wife-fulltime.tsv",
			"inflation":   "data/environment/inflation.tsv",
		}},
	}

	got := compare.Slots(subjects)

	want := &tsv.Table{
		Header: []tsv.ColumnName{"slot", "分類", "base", "wife-fulltime"},
		Rows: [][]string{
			{"income_wife", "入力",
				"data/controllable/income-wife.tsv",
				"data/controllable/scenario/income-wife-fulltime.tsv"},
		},
	}
	assertTable(t, got, want)
}

func TestClassOf(t *testing.T) {
	for _, test := range []struct {
		path string
		want compare.Class
	}{
		{"data/controllable/income-wife.tsv", compare.Chosen},
		{"data/controllable/scenario/income-wife-fulltime.tsv", compare.Chosen},
		{"data/environment/inflation.tsv", compare.Environment},
		{"data/environment/scenario/inflation-2percent.tsv", compare.Environment},

		{"actuals/balance.tsv", compare.Record},

		{"data/law/kokuho-setagaya.tsv", compare.Unclassified},
	} {
		t.Run(test.path, func(t *testing.T) {
			if got := compare.ClassOf(test.path); got != test.want {
				t.Errorf("ClassOf(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}

func TestSlotsShouldReportASlotOnlyOneProjectFills(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Paths: map[tsv.Slot]string{}},
		{Name: "with-crisis", Paths: map[tsv.Slot]string{
			"financial_crisis": "data/environment/financial-crisis.tsv",
		}},
	}

	got := compare.Slots(subjects)

	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"slot", "分類", "base", "with-crisis"},
		Rows: [][]string{
			{"financial_crisis", "環境", "", "data/environment/financial-crisis.tsv"},
		},
	})
}

func TestSlotsShouldRefuseToPickAClassWhenTheProjectsDisagree(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Paths: map[tsv.Slot]string{
			"living_cost": "data/controllable/living-cost.tsv",
		}},
		{Name: "odd", Paths: map[tsv.Slot]string{
			"living_cost": "data/environment/living-cost.tsv",
		}},
	}

	got := compare.Slots(subjects)

	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"slot", "分類", "base", "odd"},
		Rows: [][]string{
			{"living_cost", "不明",
				"data/controllable/living-cost.tsv",
				"data/environment/living-cost.tsv"},
		},
	})
}

func TestSlotsShouldNameASlotTheCommandLineSettled(t *testing.T) {
	overridden := map[tsv.Slot]bool{"inflation": true}
	subjects := []compare.Subject{
		{Name: "base", Overridden: overridden, Paths: map[tsv.Slot]string{
			"inflation": "data/environment/scenario/inflation-2percent.tsv",
		}},
		{Name: "settle-2050", Overridden: overridden, Paths: map[tsv.Slot]string{
			"inflation": "data/environment/scenario/inflation-2percent.tsv",
		}},
	}

	got := compare.Slots(subjects)

	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"slot", "分類", "base", "settle-2050"},
		Rows: [][]string{
			{"inflation", "コマンド引数",
				"data/environment/scenario/inflation-2percent.tsv",
				"data/environment/scenario/inflation-2percent.tsv"},
		},
	})
}

func TestWarningsShouldSayWhichSlotsTheCommandLineSettled(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Overridden: map[tsv.Slot]bool{"inflation": true},
			Paths: map[tsv.Slot]string{"inflation": "x.tsv"}},
		{Name: "other", Overridden: map[tsv.Slot]bool{"inflation": true},
			Paths: map[tsv.Slot]string{"inflation": "x.tsv"}},
	}

	got := compare.Warnings(subjects)

	var said bool
	for _, warning := range got {
		if strings.Contains(warning, "inflation") && strings.Contains(warning, "x.tsv") {
			said = true
		}
	}
	if !said {
		t.Errorf("差し替えた slot が報告されていない: %v", got)
	}
}
