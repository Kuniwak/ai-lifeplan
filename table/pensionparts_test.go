package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/input"
)

func TestThePensionPartsShouldAddUpToWhatIsReceived(t *testing.T) {
	built := incomeOfTheBaseProject(t, "夫", input.IncomeHusbandSlot)

	years := 0
	for _, row := range built.Rows() {
		v := row.Value
		if v.PensionReceived == 0 {
			if parts := v.PensionBasic + v.PensionProportional + v.PensionSupplement; parts != 0 {
				t.Errorf("%d: 年金収入が 0 なのに内訳が %d ある", row.Year, parts)
			}
			continue
		}
		years++

		parts := v.PensionBasic + v.PensionProportional + v.PensionSupplement
		if gap := parts - v.PensionReceived; gap < 0 || gap > 2 {
			t.Errorf("%d: 年金(基礎) %d ＋ 年金(報酬比例) %d ＋ 年金(加給) %d = %d が、"+
				"年金収入 %d と %d 円違う（切捨ての差は 0〜2 円のはず）",
				row.Year, v.PensionBasic, v.PensionProportional, v.PensionSupplement,
				parts, v.PensionReceived, gap)
		}
	}
	if years == 0 {
		t.Fatal("年金を受け取る年が 1 つも無い。この検査は何も見ていない")
	}
}
