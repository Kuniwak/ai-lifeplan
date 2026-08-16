package law

import (
	"fmt"
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const (
	theBasicPensionBase         = money.Yen(780_900)
	theSupplementaryPensionBase = money.Yen(225_000)
)

var theAnnouncedMonthlyBasicPension = map[date.Year]money.Yen{
	2025: 69_308,
	2026: 70_608,
}

func TestTheBasicPensionShouldAgreeWithTheAnnouncement(t *testing.T) {
	loaded := MustLoadBasicPensionFullAmounts(t, os.DirFS("../"+LawDirectory))

	for year, monthly := range theAnnouncedMonthlyBasicPension {
		t.Run(fmt.Sprintf("%d年度", year), func(t *testing.T) {
			got := loaded.Amount(year)

			if want := got / 12; want != monthly {
				t.Errorf("%d 年度の満額 %d 円は月額 %d 円になる。発表は %d 円である",
					year, got, want, monthly)
			}

			candidates := 0
			for annual := monthly * 12; annual < (monthly+1)*12; annual++ {
				if annual%100 == 0 {
					candidates++
					if annual != got {
						t.Errorf("月額 %d 円になる 100 円刻みの年額は %d 円のはずだが、表は %d 円である",
							monthly, annual, got)
					}
				}
			}
			if candidates != 1 {
				t.Errorf("月額 %d 円になる 100 円刻みの年額が %d 個ある。一意に決まらない", monthly, candidates)
			}

			if got <= theBasicPensionBase {
				t.Errorf("%d 年度の満額が %d 円で、基準額 %d 円を上回っていない", year, got, theBasicPensionBase)
			}
		})
	}
}

func supplementaryFor(numerator, denominator money.Yen) money.Yen {
	const hundred = int64(theHundredYenStep)
	return money.Yen(money.HalfUp(
		int64(theSupplementaryPensionBase)*int64(numerator),
		int64(denominator)*hundred) * hundred)
}

const theHundredYenStep money.Yen = 100
