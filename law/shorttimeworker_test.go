package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestWhoTheEmployeesSchemesCover(t *testing.T) {
	const notAStudent, aStudent = false, true
	at := law.Workplace{NormalWeeklyHours: 40, Specified: true}

	for _, c := range []struct {
		name    string
		hours   int
		monthly money.Yen
		student bool
		want    bool
	}{
		{"正社員", 40, 500_000, notAStudent, true},

		{"四分の三ちょうど・報酬は八万八千円未満", 30, 50_000, notAStudent, true},

		{"四分の三未満・二十時間以上・八万八千円以上", 29, 100_000, notAStudent, true},

		{"イ 二十時間未満", 19, 200_000, notAStudent, false},
		{"ロ 報酬が八万八千円未満", 20, 87_999, notAStudent, false},
		{"ロ 八万八千円ちょうどは未満ではない", 20, 88_000, notAStudent, true},
		{"ハ 学生", 25, 100_000, aStudent, false},

		{"働いていない", 0, 0, notAStudent, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := law.EmployeesInsuranceCovers(at, c.hours, c.monthly, c.student)
			if got != c.want {
				t.Errorf("週 %d 時間・報酬月額 %d 円・学生 %v が %v。%v のはず",
					c.hours, c.monthly, c.student, got, c.want)
			}
		})
	}
}

func TestAMillionAYearTurnsOnTheHoursAndNotOnTheWage(t *testing.T) {
	const monthly = money.Yen(1_000_000 / 12)
	at := law.Workplace{NormalWeeklyHours: 40, Specified: true}

	for hours := 1; hours < 30; hours++ {
		if law.EmployeesInsuranceCovers(at, hours, monthly, false) {
			t.Errorf("週 %d 時間で被保険者になった。八万八千円未満なので外れるはず", hours)
		}
	}
	if !law.EmployeesInsuranceCovers(at, 30, monthly, false) {
		t.Error("週 30 時間で被保険者にならない。四分の三に届けば報酬は読まれないはず")
	}
}

func TestEmploymentInsuranceAndTheEmployeesSchemesShouldPartAtTwentyHours(t *testing.T) {
	const monthly = money.Yen(1_000_000 / 12)
	const notAStudent = false
	at := law.Workplace{NormalWeeklyHours: 40, Specified: true}

	if law.EmployeesInsuranceCovers(at, 20, monthly, false) {
		t.Error("週 20 時間・報酬 83,000 円が健保の被保険者になった。ロで外れるはず")
	}
	if !law.EmploymentInsuranceCovers(20, notAStudent) {
		t.Error("週 20 時間で雇用保険が掛からない。第六条第一号は二十時間未満だけを外すはず")
	}
	if law.EmploymentInsuranceCovers(19, notAStudent) {
		t.Error("週 19 時間で雇用保険が掛かった。二十時間未満は適用除外のはず")
	}
	if !law.EmployeesInsuranceCovers(at, 30, monthly, false) || !law.EmploymentInsuranceCovers(30, notAStudent) {
		t.Error("週 30 時間でどちらかが掛からない")
	}
}

func TestAStudentIsOutsideEmploymentInsuranceToo(t *testing.T) {
	const aStudent = true
	at := law.Workplace{NormalWeeklyHours: 40, Specified: true}

	if law.EmploymentInsuranceCovers(40, aStudent) {
		t.Error("週 40 時間の学生に雇用保険が掛かった。第六条第四号で外れるはず")
	}
	if law.EmployeesInsuranceCovers(at, 25, 100_000, aStudent) {
		t.Error("学生が健保の被保険者になった。ハで外れるはず")
	}
}
