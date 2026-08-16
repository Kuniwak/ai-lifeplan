package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestTheChildAllowanceLimitsShouldFollowFromTheCabinetOrder(t *testing.T) {
	limitArticle := egovArticle(t, "児童手当法施行令-第1条-令和6年9月分まで.xml")
	ceilingArticle := egovArticle(t, "児童手当法施行令-第7条-令和6年9月分まで.xml")

	base := money.Yen(egovAmount(t, limitArticle, `がないときは(`+theKanjiNumeral+`)円とし`))
	ceilingBase := money.Yen(egovAmount(t, ceilingArticle, `がないときは(`+theKanjiNumeral+`)円とし`))

	step := money.Yen(egovAmount(t, limitArticle, `又は児童一人につき(`+theKanjiNumeral+`)円`))
	if got := money.Yen(egovAmount(t, ceilingArticle, `又は児童一人につき(`+theKanjiNumeral+`)円`)); got != step {
		t.Fatalf("第一条の加算が %d 円、第七条の加算が %d 円。同じはず", step, got)
	}
	if step != childAllowanceLimitStep {
		t.Errorf("1 人あたりの加算が %d 円。政令は %d 円と定めている", childAllowanceLimitStep, step)
	}

	loaded := childAllowanceTable(t)

	for dependents := range len(loaded.limits) {
		got, ok := loaded.Limits(beforeTheReform, dependents)
		if !ok {
			t.Fatalf("%d 年には限度額があるはずだ", beforeTheReform)
		}

		if want := base + step*money.Yen(dependents); got.IncomeLimit != want {
			t.Errorf("扶養親族等 %d 人の所得制限限度額が %d 円。施行令第一条は %d 円と定めている",
				dependents, got.IncomeLimit, want)
		}
		if want := ceilingBase + step*money.Yen(dependents); got.IncomeCeiling != want {
			t.Errorf("扶養親族等 %d 人の所得上限限度額が %d 円。施行令第七条は %d 円と定めている",
				dependents, got.IncomeCeiling, want)
		}
	}
}

func TestTheChildAllowanceLimitsShouldRecordTheOlderDependentProviso(t *testing.T) {
	limitArticle := egovArticle(t, "児童手当法施行令-第1条-令和6年9月分まで.xml")

	older := egovAmount(t, limitArticle,
		`老人扶養親族であるときは、当該同一生計配偶者又は老人扶養親族一人につき(`+theKanjiNumeral+`)円`)

	if want := int64(440_000); older != want {
		t.Errorf("政令の老人加算が %d 円。%d 円のはず", older, want)
	}
	if older == int64(childAllowanceLimitStep) {
		t.Error("老人加算と通常の加算が同じ額になった。表が 1 つの加算しか持たない理由が消えている")
	}
}
