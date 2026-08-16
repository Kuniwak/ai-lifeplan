package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/table"
)

func TestTheWifesTotalReceiptsShouldCountThePension(t *testing.T) {
	built := incomeOfTheBaseProject(t, "妻", input.IncomeWifeSlot)

	row, ok := built.At(date.Year(2059))
	if !ok {
		t.Fatal("2059 is missing")
	}
	if want := money.Yen(1_000_000); row.PensionReceived != want || row.Total != want {
		t.Errorf("年金収入 %d, 総収入 %d, want %d for both", row.PensionReceived, row.Total, want)
	}
}

func TestTheHaircutShouldApplyToWhatIsReceived(t *testing.T) {
	p := table.Pension{
		StartYear: 2059, Basic: 900_000, Proportional: 150_000, Supplement: 50_000,
		SupplementFrom:    date.Date{Year: 2059, Month: 1, Day: 1},
		SupplementThrough: date.Date{Year: 2059, Month: 12, Day: 31},
		Expected:          money.NewRate(70, 100),
	}

	if got, want := p.Received(2059), money.Yen(770_000); got != want {
		t.Errorf("%d 円。%d 円のはず", got, want)
	}
	if got := p.Received(2058); got != 0 {
		t.Errorf("受給開始の前の年に %d 円出ている", got)
	}
}

func TestTheSupplementShouldStopWhenTheSpouseTurnsSixtyFive(t *testing.T) {
	p := table.Pension{
		StartYear: 2055, Supplement: 423_700,
		SupplementFrom:    date.Date{Year: 2055, Month: 7, Day: 1},
		SupplementThrough: date.Date{Year: 2057, Month: 9, Day: 30},
		Expected:          money.NewRate(1, 1),
	}

	for _, c := range []struct {
		year   date.Year
		months int
	}{
		{2054, 0},
		{2055, 6},
		{2056, 12},
		{2057, 9},
		{2058, 0},
	} {
		want := money.Yen(0)
		if c.months > 0 {
			want = money.Yen(423_700 * c.months / 12)
		}
		_, _, got := p.PartsReceived(c.year)
		if got != want {
			t.Errorf("%d 年の加給年金 = %d、%d か月ぶんの %d のはず", c.year, got, c.months, want)
		}
	}
}

func TestASupplementWithNoWindowShouldPayNothing(t *testing.T) {
	p := table.Pension{StartYear: 2059, Supplement: 423_700, Expected: money.NewRate(1, 1)}
	if got := p.Received(2060); got != 0 {
		t.Errorf("窓の無い加給年金が %d 円払われた", got)
	}
}

func TestTheWifeShouldEarnNoBusinessReceiptsInThisPlan(t *testing.T) {
	for _, row := range incomeOfTheBaseProject(t, "妻", input.IncomeWifeSlot).Rows() {
		if row.Value.BusinessReceipts != 0 {
			t.Errorf("%d: 事業収入 %d; the plan gives the wife a salary, not business receipts",
				row.Year, row.Value.BusinessReceipts)
		}
	}
}
