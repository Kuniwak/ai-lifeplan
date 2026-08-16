package table

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
)

func payTable(t *testing.T, text string) *tsv.Table {
	t.Helper()

	read, err := tsv.Read(strings.NewReader(strings.TrimLeft(text, "\n")))
	if err != nil {
		t.Fatalf("tsv.Read: %v", err)
	}
	return read
}

func TestAPayTableWithBusinessIncomeShouldNameTheMissingBlueFormColumn(t *testing.T) {
	missing := payTable(t, `
西暦	給与収入[円/年]	賞与収入[円/年]	賞与回数[回/年]	育休月数[月]	事業収入[円/年]	事業必要経費[円/年]
2018	0	0	0	0	1000000	0
`)

	_, err := readPay(missing, input.IncomeWifeSlot, 2018, 2019)
	if err == nil {
		t.Fatal("a pay table with 事業収入 and no 青色申告区分 was accepted")
	}
	if want := string(input.BlueFormRecordKeepingColumn); !strings.Contains(err.Error(), want) {
		t.Errorf("the error does not name %q: %v", want, err)
	}
	if unwanted := "is not a recognised"; strings.Contains(err.Error(), unwanted) {
		t.Errorf("the error blames a value nobody wrote: %v", err)
	}
}

func TestAPayTableWithoutBusinessIncomeShouldNotDemandTheBlueFormColumn(t *testing.T) {
	salaried := payTable(t, `
西暦	給与収入[円/年]	賞与収入[円/年]	賞与回数[回/年]	育休月数[月]
2018	5000000	0	0	0
`)

	built, err := readPay(salaried, input.IncomeWifeSlot, 2018, 2019)
	if err != nil {
		t.Fatalf("readPay: %v", err)
	}
	row, ok := built.At(2018)
	if !ok {
		t.Fatal("2018 is missing")
	}
	if row.BlueFormRecordKeeping != "" {
		t.Errorf("青色申告区分 = %q, want it unset; the table does not say", row.BlueFormRecordKeeping)
	}
}
