package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
)

type DisabilityPensionEligible string

const (
	DisabilityPensionYes DisabilityPensionEligible = "はい"
	DisabilityPensionNo  DisabilityPensionEligible = "いいえ"
)

func DisabilityPensionAnswers() []DisabilityPensionEligible {
	return []DisabilityPensionEligible{DisabilityPensionYes, DisabilityPensionNo}
}

func (a DisabilityPensionEligible) Eligible() (bool, error) {
	switch a {
	case DisabilityPensionYes:
		return true, nil
	case DisabilityPensionNo:
		return false, nil
	default:
		return false, fmt.Errorf(
			"law.DisabilityPensionEligible.Eligible: %q は障害厚生年金の受給要件についての答えではない。"+
				"%q か %q を書くこと。空欄は「いいえ」ではなく、誰も答えていないという意味である",
			string(a), string(DisabilityPensionYes), string(DisabilityPensionNo))
	}
}

const (
	HealthDependantIncomeLimit      money.Yen = 1_300_000
	HealthDependantOlderIncomeLimit money.Yen = 1_800_000
	HealthDependantOlderAge                   = 60
)

func HealthDependantIncomeLimitAt(age int, disabilityPension DisabilityPensionEligible) money.Yen {
	if age >= HealthDependantOlderAge || disabilityPension == DisabilityPensionYes {
		return HealthDependantOlderIncomeLimit
	}
	return HealthDependantIncomeLimit
}

func HealthDependant(insured Cover, insuredReceipts, receipts money.Yen, age int, disabilityPension DisabilityPensionEligible) bool {
	return insured == EmployeesHealthInsurance &&
		receipts < HealthDependantIncomeLimitAt(age, disabilityPension) &&
		receipts*2 < insuredReceipts
}

func PrincipalInsured[N ~string](covers map[N]Cover, receipts map[N]money.Yen) (N, bool) {
	var principal N
	var found bool
	for name, cover := range covers {
		if cover != EmployeesHealthInsurance {
			continue
		}
		if !found || receipts[name] > receipts[principal] ||
			(receipts[name] == receipts[principal] && name < principal) {
			principal, found = name, true
		}
	}
	return principal, found
}
