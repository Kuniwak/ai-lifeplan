package yeartest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/yeartest"
)

func TestEachYearShouldReadBothSpellingsOfAYear(t *testing.T) {
	for name, c := range map[string]struct {
		rows [][]string
		walk func(yeartest.TB, *tsv.Table, func(date.Year, []string))
	}{
		"写しは 年 つきで書く":  {[][]string{{"2018年", "a"}, {"2019年", "b"}}, yeartest.EachSheetsYear},
		"出力は 年 を書かない":  {[][]string{{"2018", "a"}, {"2019", "b"}}, yeartest.EachYear},
		"写しの中で入り混じっても": {[][]string{{"2018年", "a"}, {"2019", "b"}}, yeartest.EachSheetsYear},
	} {
		t.Run(name, func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"西暦", "値"}, Rows: c.rows}

			var years []date.Year
			c.walk(t, table, func(year date.Year, _ []string) {
				years = append(years, year)
			})

			if want := []date.Year{2018, 2019}; len(years) != len(want) || years[0] != want[0] || years[1] != want[1] {
				t.Errorf("years = %v, want %v", years, want)
			}
		})
	}
}

func TestEachYearShouldWalkInTheOrderTheFileWrote(t *testing.T) {
	table := &tsv.Table{
		Header: []tsv.ColumnName{"西暦"},
		Rows:   [][]string{{"2031"}, {"2018"}, {"2025"}},
	}

	var years []date.Year
	yeartest.EachYear(t, table, func(year date.Year, _ []string) {
		years = append(years, year)
	})

	for i, want := range []date.Year{2031, 2018, 2025} {
		if years[i] != want {
			t.Errorf("%d 番目が %d である（%d のはず）", i+1, years[i], want)
		}
	}
}

func TestEachYearOfShouldTakeAnotherColumnName(t *testing.T) {
	table := &tsv.Table{
		Header: []tsv.ColumnName{"取得年", "土地[円]"},
		Rows:   [][]string{{"2022", "30000000"}},
	}

	walked := 0
	yeartest.EachYearOf(t, table, "取得年", func(year date.Year, _ []string) {
		if year != 2022 {
			t.Errorf("year = %d, want 2022", year)
		}
		walked++
	})
	if walked != 1 {
		t.Errorf("歩いた行が %d 行である（1 行のはず）", walked)
	}
}

func TestEachYearShouldRefuseTheSheetsSpelling(t *testing.T) {
	table := &tsv.Table{
		Header: []tsv.ColumnName{"西暦"},
		Rows:   [][]string{{"2018年"}},
	}

	var watched watcher
	walked := 0
	yeartest.EachYear(&watched, table, func(date.Year, []string) { walked++ })

	if watched.failure == "" {
		t.Error(`EachYear が "2018年" を受け入れた。写しの書き方を読むのは EachSheetsYear の仕事である`)
	}
	if walked != 0 {
		t.Errorf("拒んだのに %d 行を歩いた", walked)
	}

	var quiet watcher
	yeartest.EachSheetsYear(&quiet, table, func(date.Year, []string) {})
	if quiet.failure != "" {
		t.Errorf("EachSheetsYear が写しの書き方を拒んだ: %s", quiet.failure)
	}
}

func TestEachYearShouldSayWhichColumnIsMissing(t *testing.T) {
	var watched watcher
	yeartest.EachYear(&watched, &tsv.Table{Header: []tsv.ColumnName{"年度"}}, func(date.Year, []string) {})

	if !strings.Contains(watched.failure, "西暦") {
		t.Errorf("どの列が無いのかを言っていない: %q", watched.failure)
	}
}

type watcher struct{ failure string }

func (w *watcher) Helper() {}

func (w *watcher) Fatalf(format string, args ...any) {
	if w.failure == "" {
		w.failure = fmt.Sprintf(format, args...)
	}
}
