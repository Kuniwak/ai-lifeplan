package law

import (
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const theMoveInYearUnderTest = date.Year(2022)

func housingLoanTermsForTheMove(t *testing.T) HousingLoanTerms {
	t.Helper()

	loaded := MustLoadHousingLoanCredits(t, os.DirFS("../"+LawDirectory))
	terms, ok := loaded.Terms(theMoveInYearUnderTest)
	if !ok {
		t.Fatalf("%d 年入居の条件が表に無い", theMoveInYearUnderTest)
	}
	return terms
}

func TestTheHousingLoanCreditShouldAgreeWithTheStatute(t *testing.T) {
	article := egovArticle(t, "租税特別措置法-第41条.xml")
	terms := housingLoanTermsForTheMove(t)

	if want := money.Yen(egovAmount(t, article, `合計所得金額」という。）が(`+theKanjiNumeral+`)円以下である年`)); terms.IncomeLimit != want {
		t.Errorf("合計所得金額上限が %d 円。第41条第1項は %d 円と定めている", terms.IncomeLimit, want)
	}

	if want := egovAmount(t, article, `場合には、(`+theKanjiNumeral+`)年間）の適用年`); int64(terms.Years) != want {
		t.Errorf("控除期間が %d 年。第41条第1項は令和4年入居について %d 年間と定めている", terms.Years, want)
	}

	if !strings.Contains(article, "居住年が令和四年から令和十二年までの各年である場合 〇・七パーセント") {
		t.Error("第41条第4項第2号が「令和四年から令和十二年まで 〇・七パーセント」と読めない")
	}
	want, err := money.ParsePercent("0.70%")
	if err != nil {
		t.Fatalf("money.ParsePercent: %v", err)
	}
	if terms.Rate != want {
		t.Errorf("控除率が %v。第41条第4項第2号は %v と定めている", terms.Rate, want)
	}
}
