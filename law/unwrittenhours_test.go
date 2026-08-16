package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestPayWithNoHoursWrittenShouldReadAsFullTime(t *testing.T) {
	const notAStudent = false
	at := law.Workplace{NormalWeeklyHours: 40, Specified: true}

	for _, c := range []struct {
		name    string
		hours   int
		monthly money.Yen
		want    bool
	}{
		{"報酬があり時間が書かれていない", 0, 400_000, true},
		{"報酬も時間も無い（本当に働いていない）", 0, 0, false},
		{"時間が書かれていれば、そちらが決める", 19, 400_000, false},
		{"書かれた 0 時間でも報酬があれば働いている", 0, 83_000, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := law.WeeklyHoursOf(at, c.hours, c.monthly)
			covered := law.EmployeesInsuranceCovers(at, got, c.monthly, notAStudent)
			if covered != c.want {
				t.Errorf("週 %d 時間・報酬 %d 円 → 読み %d 時間 → 被保険者 %v。%v のはず",
					c.hours, c.monthly, got, covered, c.want)
			}
		})
	}
}

func TestEmploymentInsuranceShouldReadUnwrittenHoursTheSameWay(t *testing.T) {
	const notAStudent = false
	at := law.Workplace{NormalWeeklyHours: 40, Specified: true}
	if !law.EmploymentInsuranceCovers(law.WeeklyHoursOf(at, 0, 400_000), notAStudent) {
		t.Error("報酬があり時間が書かれていない年に雇用保険が立たない")
	}
	if law.EmploymentInsuranceCovers(law.WeeklyHoursOf(at, 0, 0), notAStudent) {
		t.Error("報酬も時間も無い年に雇用保険が立った")
	}
}
