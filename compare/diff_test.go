package compare_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestDiffShouldReportTheColumnAndYearThatDiffer(t *testing.T) {
	header := []tsv.ColumnName{"西暦", "総収入", "総支出"}
	subjects := []compare.Subject{
		{Name: "base", Tables: tables("timeline", header,
			[]string{"2030", "1000", "800"},
			[]string{"2031", "1000", "800"})},
		{Name: "settle-2050", Tables: tables("timeline", header,
			[]string{"2030", "1000", "800"},
			[]string{"2031", "1000", "950"})},
	}

	got, err := compare.Diff(subjects)

	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"表", "西暦", "列", "プロジェクト", "基準", "値", "差", "年の有無"},
		Rows: [][]string{
			{"timeline", "2031", "総支出", "settle-2050", "800", "950", "150", "両方"},
		},
	})
}

func TestDiffShouldBeEmptyWhenTheProjectsAgree(t *testing.T) {
	header := []tsv.ColumnName{"西暦", "総収入"}
	subjects := []compare.Subject{
		{Name: "base", Tables: tables("timeline", header, []string{"2030", "1000"})},
		{Name: "same", Tables: tables("timeline", header, []string{"2030", "1000"})},
	}

	got, err := compare.Diff(subjects)

	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(got.Rows) != 0 {
		t.Errorf("%d row(s), want none: %v", len(got.Rows), got.Rows)
	}
}

func TestDiffShouldReportAValueItCannotSubtract(t *testing.T) {
	header := []tsv.ColumnName{"西暦", "所得制限限度額"}
	subjects := []compare.Subject{
		{Name: "base", Tables: tables("child-allowance", header, []string{"2030", ""})},
		{Name: "limited", Tables: tables("child-allowance", header, []string{"2030", "8580000"})},
	}

	got, err := compare.Diff(subjects)

	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"表", "西暦", "列", "プロジェクト", "基準", "値", "差", "年の有無"},
		Rows: [][]string{
			{"child-allowance", "2030", "所得制限限度額", "limited", "", "8580000", "", "両方"},
		},
	})
}

func TestDiffShouldSubtractDecimals(t *testing.T) {
	for _, test := range []struct {
		name    string
		was, is string
		want    string
	}{
		{name: "上がる", was: "1.0000", is: "1.0040", want: "0.0040"},
		{name: "下がる", was: "1.0040", is: "1.0000", want: "-0.0040"},
		{name: "桁数が違えば長いほうに合わせる", was: "1.0", is: "1.0040", want: "0.0040"},
		{name: "整数は整数のまま", was: "800", is: "950", want: "150"},
	} {
		t.Run(test.name, func(t *testing.T) {
			header := []tsv.ColumnName{"西暦", "物価"}
			subjects := []compare.Subject{
				{Name: "base", Tables: tables("real", header, []string{"2030", test.was})},
				{Name: "other", Tables: tables("real", header, []string{"2030", test.is})},
			}

			got, err := compare.Diff(subjects)

			if err != nil {
				t.Fatalf("Diff: %v", err)
			}
			if len(got.Rows) != 1 {
				t.Fatalf("%d row(s), want 1: %v", len(got.Rows), got.Rows)
			}
			if got.Rows[0][6] != test.want {
				t.Errorf("差 = %q, want %q", got.Rows[0][6], test.want)
			}
		})
	}
}

func TestDiffShouldCompareTheYearsInCommonWhenTheRecordsDifferInLength(t *testing.T) {
	header := []tsv.ColumnName{"西暦", "総収入"}
	diffHeader := []tsv.ColumnName{"表", "西暦", "列", "プロジェクト", "基準", "値", "差", "年の有無"}

	for _, test := range []struct {
		name     string
		subjects []compare.Subject
		want     [][]string
	}{
		{
			name: "基準のほうが長い",
			subjects: []compare.Subject{
				{Name: "base", Tables: tables("outturn", header,
					[]string{"2030", "1000"},
					[]string{"2031", "1000"})},
				{Name: "as-2022", Tables: tables("outturn", header,
					[]string{"2031", "950"})},
			},
			want: [][]string{
				{"outturn", "2030", "", "as-2022", "", "", "", "基準のみ"},
				{"outturn", "2031", "総収入", "as-2022", "1000", "950", "-50", "両方"},
			},
		},
		{
			name: "相手のほうが長い",
			subjects: []compare.Subject{
				{Name: "base", Tables: tables("outturn", header, []string{"2030", "1000"})},
				{Name: "longer", Tables: tables("outturn", header,
					[]string{"2030", "1000"},
					[]string{"2031", "1000"})},
			},
			want: [][]string{
				{"outturn", "2031", "", "longer", "", "", "", "プロジェクトのみ"},
			},
		},
		{
			name: "3 つ並べても、行がどのプロジェクトの話かは取り違えない",
			subjects: []compare.Subject{
				{Name: "base", Tables: tables("outturn", header, []string{"2030", "1000"})},
				{Name: "shorter", Tables: tables("outturn", header)},
				{Name: "longer", Tables: tables("outturn", header,
					[]string{"2030", "1000"},
					[]string{"2031", "1100"})},
			},
			want: [][]string{
				{"outturn", "2030", "", "shorter", "", "", "", "基準のみ"},
				{"outturn", "2031", "", "longer", "", "", "", "プロジェクトのみ"},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := compare.Diff(test.subjects)

			if err != nil {
				t.Fatalf("Diff: %v", err)
			}
			assertTable(t, got, &tsv.Table{Header: diffHeader, Rows: test.want})
		})
	}
}

func TestDiffShouldOrderTheRowsByYearAndNotByItsText(t *testing.T) {
	header := []tsv.ColumnName{"西暦", "総収入"}
	subjects := []compare.Subject{
		{Name: "base", Tables: tables("outturn", header,
			[]string{"1000", "1"},
			[]string{"999", "1"},
			[]string{"2030", "1"})},
		{Name: "other", Tables: tables("outturn", header,
			[]string{"1000", "2"},
			[]string{"999", "2"},
			[]string{"2030", "2"})},
	}

	got, err := compare.Diff(subjects)

	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	var years []string
	for _, row := range got.Rows {
		years = append(years, row[1])
	}
	want := []string{"999", "1000", "2030"}
	if len(years) != len(want) {
		t.Fatalf("years = %v, want %v", years, want)
	}
	for i, year := range want {
		if years[i] != year {
			t.Fatalf("years = %v, want %v", years, want)
		}
	}
}

func TestDiffShouldRefuseATableWithTheSameYearTwice(t *testing.T) {
	header := []tsv.ColumnName{"西暦", "総収入"}
	subjects := []compare.Subject{
		{Name: "base", Tables: tables("outturn", header,
			[]string{"2030", "1000"},
			[]string{"2030", "1100"})},
		{Name: "other", Tables: tables("outturn", header, []string{"2030", "1000"})},
	}

	_, err := compare.Diff(subjects)

	if err == nil {
		t.Fatal("Diff accepted a table with the same year twice")
	}
	if !strings.Contains(err.Error(), "2030") {
		t.Errorf("%q does not name %q", err, "2030")
	}
}

func TestDiffShouldRefuseTablesOfDifferentShape(t *testing.T) {
	for _, test := range []struct {
		name  string
		other *tsv.Table
		says  string
	}{
		{
			name:  "列が違う",
			other: &tsv.Table{Header: []tsv.ColumnName{"西暦", "総収入", "総支出"}, Rows: [][]string{{"2030", "1000", "800"}}},
			says:  "総支出",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			subjects := []compare.Subject{
				{Name: "base", Tables: tables("timeline",
					[]tsv.ColumnName{"西暦", "総収入"}, []string{"2030", "1000"})},
				{Name: "other", Tables: map[plan.TableName]*tsv.Table{"timeline": test.other}},
			}

			_, err := compare.Diff(subjects)

			if err == nil {
				t.Fatal("Diff accepted tables of different shape")
			}
			if !strings.Contains(err.Error(), test.says) {
				t.Errorf("%q does not name %q", err, test.says)
			}
		})
	}
}
