package law

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

func TestTheSpouseIncomeCeilingShouldMoveOnTheYearsTheLawMovedIt(t *testing.T) {
	table, err := LoadSpouseIncomeCeilingTable(os.DirFS("../" + LawDirectory))
	if err != nil {
		t.Fatalf("law.LoadSpouseIncomeCeilingTable: %v", err)
	}

	cases := map[int]money.Yen{
		2018: 380_000,
		2019: 380_000,
		2020: 480_000,
		2024: 480_000,
		2025: 580_000,
		2090: 580_000,
	}

	for year, want := range cases {
		if got := table.Ceiling(date.Year(year)); got != want {
			t.Errorf("%d 年分の所得要件が %v である（%v のはず）", year, got, want)
		}
	}
}

func TestTheCeilingShouldDecideWhetherASpouseCountsAtAll(t *testing.T) {
	table, err := LoadSpouseIncomeCeilingTable(os.DirFS("../" + LawDirectory))
	if err != nil {
		t.Fatalf("law.LoadSpouseIncomeCeilingTable: %v", err)
	}

	const inTheBand money.Yen = 500_000

	if inTheBand <= table.Ceiling(2024) {
		t.Error("令和6年分で 50万 が 同一生計配偶者 の範囲に入っている（48万 が上限のはず）")
	}
	if inTheBand > table.Ceiling(2025) {
		t.Error("令和7年分で 50万 が 同一生計配偶者 の範囲から外れている（58万 が上限のはず）")
	}
}
