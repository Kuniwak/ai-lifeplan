package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/panictest"
)

func TestAShortTimeWorkerOutsideASpecifiedWorkplaceIsNotCovered(t *testing.T) {
	const notAStudent = false
	const monthly = 100_000

	specified := law.Workplace{NormalWeeklyHours: 40, Specified: true}
	notSpecified := law.Workplace{NormalWeeklyHours: 40, Specified: false}

	if !law.EmployeesInsuranceCovers(specified, 25, monthly, notAStudent) {
		t.Error("特定適用事業所の週 25 時間が被保険者になっていない")
	}
	if law.EmployeesInsuranceCovers(notSpecified, 25, monthly, notAStudent) {
		t.Error("特定適用事業所でない事業所の週 25 時間が被保険者になっている。" +
			"附則第四十六条第一項は四分の三未満というだけで外す")
	}

	if !law.EmployeesInsuranceCovers(notSpecified, 30, monthly, notAStudent) {
		t.Error("四分の三に届いた人が事業所の規模で外された")
	}
}

func TestNormalWeeklyHoursShouldComeFromTheWorkplace(t *testing.T) {
	const notAStudent = false
	const monthly = 87_000

	thirtyTwo := law.Workplace{NormalWeeklyHours: 32, Specified: true}
	forty := law.Workplace{NormalWeeklyHours: 40, Specified: true}

	if law.EmployeesInsuranceCovers(forty, 24, monthly, notAStudent) {
		t.Error("通常の労働者が週 40 時間なら、週 24 時間はロで外れる")
	}
	if !law.EmployeesInsuranceCovers(thirtyTwo, 24, monthly, notAStudent) {
		t.Error("通常の労働者が週 32 時間なら、週 24 時間は四分の三ちょうどで被保険者である")
	}
}

func TestAWorkplaceWithNoNormalWeekShouldBeRefused(t *testing.T) {
	const notAStudent = false

	refused := panictest.Recovered(func() {
		law.EmployeesInsuranceCovers(law.Workplace{}, 1, 100_000, notAStudent)
	})
	if refused == nil {
		t.Fatal("通常の労働者の週が書かれていない事業所が黙って通った。その事業所は誰も短時間労働者にしない")
	}
}
