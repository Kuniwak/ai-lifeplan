package law_test

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestSellForCashShouldNeverLeaveTheHouseholdShort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		held := money.Yen(rapid.Int64Range(1, 5_000_000_000).Draw(t, "held"))
		basis := money.Yen(rapid.Int64Range(0, int64(held)).Draw(t, "basis"))
		need := money.Yen(rapid.Int64Range(1, int64(held)).Draw(t, "need"))

		sold, tax := law.SellForCash(need, held, basis)

		if sold > held {
			t.Fatalf("残高 %d より多い %d を売った", held, sold)
		}
		if tax < 0 || sold < 0 {
			t.Fatalf("売った額 %d / 税 %d が負である", sold, tax)
		}

		if got := sold - tax; sold < held && (got < need || got > need+2) {
			t.Fatalf("手取り %d が必要額 %d をわずかに上回っていない（売 %d / 税 %d）",
				got, need, sold, tax)
		}
	})
}

func TestGainOnShouldNotOverflow(t *testing.T) {
	const huge money.Yen = 5_000_000_000

	if got := law.GainOn(huge, huge, huge); got != huge {
		t.Errorf("全額が利益なら按分は全額のはずが %d", got)
	}
	if got := law.GainOn(huge/2, huge/2, huge); got != huge/4 {
		t.Errorf("半分の半分は四分の一のはずが %d", got)
	}
}

func TestTsumitateNISAMaturity(t *testing.T) {
	cases := map[date.Year]date.Year{
		2018: 2037,
		2023: 2042,
	}
	for boughtIn, want := range cases {
		if got := law.TsumitateNISAMaturityOf(boughtIn); got != want {
			t.Errorf("%d 年買付の非課税期間が %d 年に終わることになっている（%d のはず）",
				boughtIn, got, want)
		}
	}
}

func TestNISAMaturityBasisShouldBeTheValueOnTheWayOut(t *testing.T) {
	const marketValue money.Yen = 622_000
	if got := law.NISAMaturityBasis(marketValue); got != marketValue {
		t.Errorf("取得価額が %v になった（%v のはず）", got, marketValue)
	}
	if marketValue-law.NISAMaturityBasis(marketValue) != 0 {
		t.Error("払い出した瞬間に含み益が立っている")
	}
}
