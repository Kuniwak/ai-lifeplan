package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestAscendingShouldFindWhatIsOutOfOrderOrRepeated(t *testing.T) {
	type testCase struct {
		Name  string
		Rows  [][]string
		Wants [][]string
	}
	cases := []testCase{
		{
			Name: "並んでいる",
			Rows: [][]string{{"3歳未満", "0"}, {"幼稚園", "3"}, {"小学校", "6"}},
		},
		{
			Name:  "同じ値が二度",
			Rows:  [][]string{{"幼稚園", "3"}, {"小学校", "3"}},
			Wants: [][]string{{"3", "二度"}},
		},
		{
			Name:  "降順",
			Rows:  [][]string{{"小学校", "6"}, {"中学校", "1"}},
			Wants: [][]string{{"1", "6"}},
		},
		{
			Name:  "数でない",
			Rows:  [][]string{{"幼稚園", "さん"}},
			Wants: [][]string{{"さん"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"就学段階", "開始満年齢"}, Rows: tc.Rows}

			got := check(t, Ascending("schooling", "開始満年齢"), "schooling", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestAscendingShouldReportAMissingColumnRatherThanCrash(t *testing.T) {
	table := &tsv.Table{Header: []tsv.ColumnName{"別の列"}, Rows: [][]string{{"x"}}}

	got := check(t, Ascending("s", "開始満年齢"), "s", table)

	assertFindings(t, got, []string{"開始満年齢"})
}
