package law_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

const (
	spouseSupplement        money.Yen = 243_800
	spouseSupplementSpecial money.Yen = 179_900
)

func TestTheSpouseSupplementShouldBeTheSumTheOfficialTableGives(t *testing.T) {
	const official money.Yen = 423_700

	got := law.SpouseSupplementaryPension(spouseSupplement, spouseSupplementSpecial, law.SupplementaryPensionMonths)
	if got != official {
		t.Errorf("加給年金額 = %d、日本年金機構の表は %d", got, official)
	}
}

func TestNineteenYearsShouldBuyNoSupplement(t *testing.T) {
	got := law.SpouseSupplementaryPension(spouseSupplement, spouseSupplementSpecial, law.SupplementaryPensionMonths-1)
	if got != 0 {
		t.Errorf("239 か月で加給年金 %d が付いた。20 年に足りない", got)
	}
}

func TestASpouseWithTwentyYearsOfHerOwnShouldSuspendIt(t *testing.T) {
	if law.SpouseSupplementaryPensionSuspended(law.SupplementaryPensionMonths - 1) {
		t.Error("239 か月の配偶者で支給停止になった")
	}
	if !law.SpouseSupplementaryPensionSuspended(law.SupplementaryPensionMonths) {
		t.Error("240 か月の配偶者で支給停止にならなかった")
	}
}

func TestTheSupplementShouldStopTheMonthTheSpouseTurnsSixtyFive(t *testing.T) {
	got := law.SpouseSupplementaryPensionThrough(date.Date{Year: 1992, Month: 9, Day: 20})
	if got.Year != 2057 || got.Month != 9 {
		t.Errorf("最後の月 = %s、2057 年 9 月のはず", got)
	}
}

func TestTheSupplementShouldStartTheMonthAfterThePensionBecomesPayable(t *testing.T) {
	payable := date.Date{Year: 2055, Month: 7, Day: 1}
	got := law.SpouseSupplementaryPensionFrom(payable)
	if got.Year != 2055 || got.Month != 8 {
		t.Errorf("最初の月 = %s、2055 年 8 月のはず", got)
	}

	if got := law.SpouseSupplementaryPensionFrom(date.Date{Year: 2055, Month: 12, Day: 1}); got.Year != 2056 || got.Month != 1 {
		t.Errorf("12 月の翌月 = %s、2056 年 1 月のはず", got)
	}
}

func TestABirthBeforeTheLastSpecialAdditionRowShouldBeRefused(t *testing.T) {
	err := law.AssertSupplementaryPensionSpecialAddition(date.Date{Year: 1943, Month: 4, Day: 1})
	if err == nil {
		t.Fatal("昭和18年4月1日生まれが通った。特別加算額の行が違う")
	}
	if !strings.Contains(err.Error(), "特別加算額") {
		t.Errorf("エラーがそのことを言っていない: %v", err)
	}
	if err := law.AssertSupplementaryPensionSpecialAddition(date.Date{Year: 1943, Month: 4, Day: 2}); err != nil {
		t.Errorf("昭和18年4月2日生まれが断られた: %v", err)
	}
}
