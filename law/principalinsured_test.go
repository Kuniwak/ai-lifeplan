package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

type person string

func TestThePrincipalInsuredIsTheEmployeeWhoEarnsTheMost(t *testing.T) {
	covers := map[person]law.Cover{
		"夫": law.EmployeesHealthInsurance,
		"妻": law.EmployeesHealthInsurance,
	}
	receipts := map[person]money.Yen{"夫": 5_000_000, "妻": 12_000_000}

	got, ok := law.PrincipalInsured(covers, receipts)
	if !ok || got != "妻" {
		t.Errorf("%q（%v）。稼ぎの大きい 妻 のはず", got, ok)
	}
}

func TestOnlySomebodyInTheEmployersHealthSchemeCanBeDependedOn(t *testing.T) {
	covers := map[person]law.Cover{
		"夫": law.NationalHealthInsurance,
		"妻": law.EmployeesHealthInsurance,
		"子": law.LateElderlyHealthCare,
	}
	receipts := map[person]money.Yen{"夫": 12_000_000, "妻": 3_000_000, "子": 0}

	got, ok := law.PrincipalInsured(covers, receipts)
	if !ok || got != "妻" {
		t.Errorf("%q（%v）。健保にいる 妻 のはず", got, ok)
	}
}

func TestThereIsNoPrincipalInsuredWhenNobodyIsInAnEmployersScheme(t *testing.T) {
	covers := map[person]law.Cover{"夫": law.NationalHealthInsurance, "妻": law.NoCover}

	if got, ok := law.PrincipalInsured(covers, map[person]money.Yen{}); ok {
		t.Errorf("被保険者がいないのに %q を返した", got)
	}
}

func TestATieIsBrokenByNameSoThatTheAnswerDoesNotMoveBetweenRuns(t *testing.T) {
	covers := map[person]law.Cover{
		"あ": law.EmployeesHealthInsurance,
		"い": law.EmployeesHealthInsurance,
		"う": law.EmployeesHealthInsurance,
	}
	receipts := map[person]money.Yen{"あ": 4_000_000, "い": 4_000_000, "う": 4_000_000}

	for i := 0; i < 200; i++ {
		got, ok := law.PrincipalInsured(covers, receipts)
		if !ok || got != "あ" {
			t.Fatalf("%d 回目に %q。名前順の先頭 あ のはず", i, got)
		}
	}
}
