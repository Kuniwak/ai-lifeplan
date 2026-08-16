package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestCrossingTheDependantWallShouldCostMoreThanItEarns(t *testing.T) {
	const age = 59
	limit := HealthDependantIncomeLimitAt(age, DisabilityPensionNo)
	if want := money.Yen(1_300_000); limit != want {
		t.Fatalf("%d歳の被扶養者の限は %d、%d のはず", age, limit, want)
	}

	const insuredReceipts money.Yen = 10_000_000

	if !HealthDependant(EmployeesHealthInsurance, insuredReceipts, limit-1, age, DisabilityPensionNo) {
		t.Errorf("収入 %d は限の 1 円下なので被扶養者のはず", limit-1)
	}
	if HealthDependant(EmployeesHealthInsurance, insuredReceipts, limit, age, DisabilityPensionNo) {
		t.Errorf("収入 %d は限ちょうどなので被扶養者から外れるはず", limit)
	}
}

func TestTheNationalPensionAloneShouldOutweighTheFirstYenOverTheWall(t *testing.T) {
	const fiscalYear = 2025
	premiums := nationalPensionTable(t)

	monthly := premiums.Monthly(fiscalYear)
	yearly := monthly * 12

	if yearly < 100_000 || yearly > 500_000 {
		t.Errorf("%d年度の国民年金保険料が年 %d 円で、十万円の桁から外れている", fiscalYear, yearly)
	}
	t.Logf("%d年度: 月 %d 円、年 %d 円", fiscalYear, monthly, yearly)
}
