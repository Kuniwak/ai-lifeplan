package law

import (
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

func TestTheForestTaxShouldAgreeWithTheStatute(t *testing.T) {
	act := egovArticle(t, "森林環境税法.xml")
	loaded, err := LoadForestEnvironmentTaxTable(os.DirFS("../" + LawDirectory))
	if err != nil {
		t.Fatalf("law.LoadForestEnvironmentTaxTable: %v", err)
	}

	rate := money.Yen(egovAmount(t, act, `森林環境税の税率は、(`+theKanjiNumeral+`)円とする`))

	if !strings.Contains(act, "第二章の規定は、令和六年度以後の年度分の森林環境税について適用する") {
		t.Error("附則第2条が「令和六年度以後の年度分」と読めない")
	}

	const theFirstYearItIsCharged = date.Year(2024)
	if got := loaded.Amount(theFirstYearItIsCharged); got != rate {
		t.Errorf("%d 年度の森林環境税が %d 円。第5条は %d 円と定めている", theFirstYearItIsCharged, got, rate)
	}
	if got := loaded.Amount(theFirstYearItIsCharged - 1); got != 0 {
		t.Errorf("%d 年度の森林環境税が %d 円。附則第2条により 0 円のはず", theFirstYearItIsCharged-1, got)
	}
}
