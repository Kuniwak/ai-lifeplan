package tsv_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestRowsByYear(t *testing.T) {
	const context = "income_husband"

	for name, c := range map[string]struct {
		table    *tsv.Table
		column   tsv.ColumnName
		years    []date.Year
		mentions []string
	}{
		"1 年 1 行": {
			table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "給与収入[円/年]"},
				Rows:   [][]string{{"2018", "8,080,000"}, {"2031", "0"}},
			},
			column: "西暦",
			years:  []date.Year{2018, 2031},
		},
		"鍵の列は 西暦 とはかぎらない": {
			table: &tsv.Table{
				Header: []tsv.ColumnName{"取得年", "土地[円]"},
				Rows:   [][]string{{"2022", "30,000,000"}},
			},
			column: "取得年",
			years:  []date.Year{2022},
		},
		"同じ年が二度": {
			table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "給与収入[円/年]"},
				Rows:   [][]string{{"2018", "1"}, {"2018", "2"}},
			},
			column:   "西暦",
			mentions: []string{"income_husband", "2018", "written twice", "one row per year"},
		},
		"年の列が無い": {
			table: &tsv.Table{
				Header: []tsv.ColumnName{"項目", "金額[円]"},
				Rows:   [][]string{{"給与収入", "1"}},
			},
			column:   "西暦",
			mentions: []string{"income_husband", "西暦"},
		},
		"年として読めない": {
			table: &tsv.Table{
				Header: []tsv.ColumnName{"西暦", "額"},
				Rows:   [][]string{{"平成30年", "1"}},
			},
			column:   "西暦",
			mentions: []string{"income_husband", "row 1", "西暦"},
		},
		"行が無い": {
			table:  &tsv.Table{Header: []tsv.ColumnName{"西暦", "額"}},
			column: "西暦",
			years:  nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			byYear, err := c.table.RowsByYear(context, c.column)

			if len(c.mentions) > 0 {
				if err == nil {
					t.Fatalf("受け入れられた: %v", byYear)
				}
				for _, want := range c.mentions {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("エラーが %q に触れていない: %v", want, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("RowsByYear: %v", err)
			}
			if len(byYear) != len(c.years) {
				t.Fatalf("%d 年ぶん返った（%d 年ぶんのはず）: %v", len(byYear), len(c.years), byYear)
			}
			for _, year := range c.years {
				if _, ok := byYear[year]; !ok {
					t.Errorf("%d 年が無い", year)
				}
			}
		})
	}
}
