package law

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
)

var theSameLivelihoodSpousePattern = `配偶者 居住者の配偶者でその居住者と生計を一にするもの.*?合計所得金額が(` +
	theKanjiNumeral + `)円以下である者をいう`

func TestTheSpouseIncomeCeilingShouldAgreeWithTheStatute(t *testing.T) {
	type testCase struct {
		Article string
		Year    date.Year
	}
	testCases := map[string]testCase{
		"令和元年分以前":        {Article: "所得税法-第2条-令和元年分.xml", Year: 2019},
		"令和2年分から令和6年分まで": {Article: "所得税法-第2条-令和6年分.xml", Year: 2024},
		"令和7年分以降":        {Article: "所得税法-第2条.xml", Year: 2025},
	}

	ceilings := spouseIncomeCeilings(t)

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			article := egovArticle(t, tc.Article)

			got := ceilings.Ceiling(tc.Year)

			if want := egovAmount(t, article, theSameLivelihoodSpousePattern); int64(got) != want {
				t.Errorf("%d 年分の所得要件が %d 円。%s は %d 円と定めている",
					tc.Year, got, tc.Article, want)
			}
		})
	}
}

func TestTheSpouseIncomeCeilingShouldReachBackToTheStartOfTheRecord(t *testing.T) {
	floor := egovArticle(t, "所得税法-第2条-平成29年4月.xml")

	if want, got := egovAmount(t, floor, theSameLivelihoodSpousePattern), int64(380_000); want != got {
		t.Errorf("記録の下限（平成29年4月）の所得要件が %d 円。表の最初の行は %d 円である", want, got)
	}
	if strings.Contains(floor, "同一生計配偶者") {
		t.Error("平成29年4月の版に 同一生計配偶者 がある。改名は平成30年分からのはず")
	}
	if !strings.Contains(floor, "控除対象配偶者") {
		t.Error("平成29年4月の版に 控除対象配偶者 が無い")
	}
}

func TestTheSpouseIncomeCeilingShouldRefuseTheYearsBeforeTheRecord(t *testing.T) {
	ceilings := spouseIncomeCeilings(t)

	if refused := panictest.Recovered(func() { _ = ceilings.Ceiling(2016) }); refused == nil {
		t.Error("2016 年に答えてしまった。**e-Gov が遡れるのは平成29年4月までで、" +
			"それより前をこのリポジトリの誰も読んでいない**")
	}

	if got := ceilings.Ceiling(2018); got != 380_000 {
		t.Errorf("2018 年分の所得要件が %d 円。380,000 円のはず", got)
	}
}
