package plan_test

import (
	"github.com/Kuniwak/lifeplan/date"
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/money"
)

func theRepaymentHistory(t *testing.T) (paid, balance map[date.Year]money.Yen) {
	t.Helper()

	history, err := actuals.ReadRepaymentsUnder("..")
	if err != nil {
		t.Fatalf("actuals.ReadRepaymentsUnder: %v", err)
	}

	paid, balance = map[date.Year]money.Yen{}, map[date.Year]money.Yen{}
	for year, sum := range history {
		if !sum.Whole {
			continue
		}
		paid[year], balance[year] = sum.Paid, sum.Balance
	}
	if len(paid) == 0 {
		t.Fatal("まるごと揃っている年が 1 年も無い")
	}
	return paid, balance
}
