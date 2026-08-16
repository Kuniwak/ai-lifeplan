package table_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestReadingARateShouldRefuseRowsThatAreNotInYearOrder(t *testing.T) {
	tables := map[tsv.Slot]*tsv.Table{
		input.InvestmentReturnSlot: {
			Header: []tsv.ColumnName{input.YearColumn, input.ReturnColumn},
			Rows: [][]string{
				{"2018", "3%"},
				{"2020", "5%"},
				{"2019", "1%"},
			},
		},
	}

	_, err := table.ReturnsFrom(tables, 2018, 2020)
	if err == nil {
		t.Fatalf("table.ReturnsFrom: 年の順に並んでいない表を受け入れた")
	}
	if !strings.Contains(err.Error(), "2019") {
		t.Errorf("table.ReturnsFrom: 順序を破っている年に触れていない: %v", err)
	}
}
