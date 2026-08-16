package compare_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/tsv"
)

var summaryHeader = []tsv.ColumnName{
	"プロジェクト", "起点", "資産が尽きる年", "不足の累計",
	"生涯総収入", "生涯総支出", "生涯総税額", "最終年", "最終年の資産合計",
}

func TestSummary(t *testing.T) {
	for _, test := range []struct {
		name    string
		subject compare.Subject
		want    []string
	}{
		{
			name: "持つ計画は、尽きる年を空欄にする",
			subject: compare.Subject{Name: "base", StartsAfter: 2029, Tables: twoYears(
				[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
				[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
			want: []string{"base", "2029", "", "0", "210", "160", "18", "2031", "1030"},
		},
		{
			name: "尽きる計画は、最初に不足した年を言う",
			subject: compare.Subject{Name: "depleted", StartsAfter: 2029, Tables: twoYears(
				[2]string{"100", "110"}, [2]string{"80", "900"}, [2]string{"20", "-790"},
				[2]string{"1000", "0"}, [2]string{"0", "60"}, [2]string{"9", "9"})},
			want: []string{"depleted", "2029", "2031", "60", "210", "980", "18", "2031", "0"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			subjects := []compare.Subject{test.subject}

			got, err := compare.Summary(subjects)

			if err != nil {
				t.Fatalf("Summary: %v", err)
			}
			assertTable(t, got, &tsv.Table{
				Header: summaryHeader,
				Rows:   [][]string{test.want},
			})
		})
	}
}

func TestSummaryShouldKeepOneRowPerProjectInOrder(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "second", StartsAfter: 2029, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
		{Name: "first", StartsAfter: 2029, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.Summary(subjects)

	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("%d row(s), want 2", len(got.Rows))
	}
	for i, want := range []string{"second", "first"} {
		if got.Rows[i][0] != want {
			t.Errorf("row %d is %q, want %q", i, got.Rows[i][0], want)
		}
	}
}
