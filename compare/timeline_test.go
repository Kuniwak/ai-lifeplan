package compare_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestTimelineShouldPutTheSameMetricOfEachProjectSideBySide(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
		{Name: "other", Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"90", "90"}, [2]string{"10", "20"},
			[2]string{"1000", "1020"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.Timeline(subjects)

	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	want := []tsv.ColumnName{
		"西暦",
		"総収入:base", "総収入:other",
		"総支出:base", "総支出:other",
		"収支:base", "収支:other",
		"資産合計:base", "資産合計:other",
		"手が届く額:base", "手が届く額:other",
		"不足:base", "不足:other",
	}
	if len(got.Header) != len(want) {
		t.Fatalf("header = %v, want %v", got.Header, want)
	}
	for i, name := range want {
		if got.Header[i] != name {
			t.Fatalf("header = %v, want %v", got.Header, want)
		}
	}
	assertRow(t, got, 0, []string{"2030", "100", "100", "80", "90", "20", "10", "1000", "1000", "1000", "1000", "0", "0"})
	assertRow(t, got, 1, []string{"2031", "110", "110", "80", "90", "30", "20", "1030", "1020", "1030", "1020", "0", "0"})
}

func TestTimelineShouldRefuseAProjectMissingAHeadlineColumn(t *testing.T) {
	tables := twoYears(
		[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
		[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	tables["assets"] = &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "資産合計", "手が届く額"},
		Rows:   [][]string{{"2030", "1000", "1000"}, {"2031", "1030", "1030"}},
	}
	subjects := []compare.Subject{{Name: "base", Tables: tables}}

	_, err := compare.Timeline(subjects)

	if err == nil {
		t.Fatal("Timeline accepted a project without 不足")
	}
	if !strings.Contains(err.Error(), "不足") {
		t.Errorf("%q does not name the missing column", err)
	}
}

func TestTimelineShouldKeepTheYearsOnlyOneProjectHas(t *testing.T) {
	base := twoYears(
		[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
		[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	longer := twoYears(
		[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
		[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	for _, name := range []plan.TableName{"timeline", "assets", "tax"} {
		extra := make([]string, len(longer[name].Header))
		extra[0] = "2032"
		if name == "assets" {
			extra = []string{"2032", "0", "0", "1000000"}
		}
		longer[name].Rows = append(longer[name].Rows, extra)
	}

	subjects := []compare.Subject{
		{Name: "base", Tables: base},
		{Name: "longer", Tables: longer},
	}

	got, err := compare.Timeline(subjects)

	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	if len(got.Rows) != 3 {
		t.Fatalf("%d row(s), want 3: %v", len(got.Rows), got.Rows)
	}
	assertRow(t, got, 2, []string{"2032", "", "", "", "", "", "", "", "0", "", "0", "", "1000000"})
}

func TestTimelineShouldNotSetAValueBesideAnotherYearsValue(t *testing.T) {
	base := twoYears(
		[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
		[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	shifted := twoYears(
		[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
		[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	for _, name := range []plan.TableName{"timeline", "assets", "tax"} {
		shifted[name].Rows[1][0] = "2033"
	}

	subjects := []compare.Subject{
		{Name: "base", Tables: base},
		{Name: "shifted", Tables: shifted},
	}

	got, err := compare.Timeline(subjects)

	if err != nil {
		t.Fatalf("Timeline: %v", err)
	}
	want := []string{"2030", "2031", "2033"}
	if len(got.Rows) != len(want) {
		t.Fatalf("%d row(s), want %d: %v", len(got.Rows), len(want), got.Rows)
	}
	for i, year := range want {
		if got.Rows[i][0] != year {
			t.Fatalf("行 %d の年 = %q, want %q", i, got.Rows[i][0], year)
		}
	}
	assertRow(t, got, 1, []string{"2031", "110", "", "80", "", "30", "", "1030", "", "1030", "", "0", ""})
	assertRow(t, got, 2, []string{"2033", "", "110", "", "80", "", "30", "", "1030", "", "1030", "", "0"})
}
