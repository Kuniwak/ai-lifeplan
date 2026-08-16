package plan_test

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/table"
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/plan"
)

func pensionOf(t *testing.T, p *plan.Plan, year date.Year) money.Yen {
	t.Helper()

	var total money.Yen
	for _, person := range []table.PersonName{plan.Earner, plan.Spouse} {
		row, ok := p.Income[person].At(year)
		if !ok {
			t.Fatalf("%d に %s の収入がありません", year, person)
		}
		total += row.PensionReceived
	}
	return total
}
