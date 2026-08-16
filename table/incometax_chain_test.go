package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func dependantsOnTheReturn(r map[string]money.Yen) int {
	credit, ok := r["定額減税"]
	if !ok {
		return 0
	}
	return int(credit/law.SpecialTaxCreditPerPerson) - 1
}

func reReDeducted(r map[string]money.Yen) money.Yen {
	if amount, ok := r["再々差引所得税額"]; ok {
		return amount
	}
	return r["差引所得税額"]
}

func taxReturnsForTheChain(t *testing.T) map[date.Year]actuals.TaxReturn {
	t.Helper()

	byYear, err := actuals.ReadTaxReturnsUnder("..")
	if err != nil {
		t.Fatalf("actuals.ReadTaxReturnsUnder: %v", err)
	}
	return byYear
}
