package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestTheEmployersAssumptionsShouldNotBeReadByAnyPlanTodayIsTheOnlyReasonTheyStand(t *testing.T) {
	const normalWeek = 40
	at := Workplace{NormalWeeklyHours: normalWeek, Specified: true}

	for name, c := range map[string]struct {
		hours     int
		monthly   money.Yen
		decidedBy string
	}{
		"夫（週 40 時間）": {40, 1_000_000, "四分の三に届く"},

		"妻（週 20 時間・月 83,000 円）": {20, 83_000, "ロで外れる"},

		"四分の三ちょうど": {30, 83_000, "四分の三に届く"},
	} {
		t.Run(name, func(t *testing.T) {
			switch c.decidedBy {
			case "四分の三に届く":
				if c.hours*4 < normalWeek*3 {
					t.Fatalf("週 %d 時間は四分の三に届かない。この行の前提が違う", c.hours)
				}
				if !EmployeesInsuranceCovers(at, c.hours, c.monthly, false) {
					t.Errorf("週 %d 時間で被保険者にならない", c.hours)
				}
			case "ロで外れる":
				if c.monthly >= ShortTimeRemunerationFloor {
					t.Fatalf("月 %d 円はロで外れない。この行の前提が違う", c.monthly)
				}
				if EmployeesInsuranceCovers(at, c.hours, c.monthly, false) {
					t.Errorf("月 %d 円はロで外れるはずが、被保険者になった", c.monthly)
				}
			}
		})
	}

	const inTheGap = 25
	if inTheGap*4 >= normalWeek*3 {
		t.Fatalf("週 %d 時間が四分の三に届いてしまう", inTheGap)
	}
	if !EmployeesInsuranceCovers(at, inTheGap, ShortTimeRemunerationFloor, false) {
		t.Error("週 25 時間・月 88,000 円で被保険者にならない。ここが前提の効く帯のはず")
	}
	if EmployeesInsuranceCovers(at, inTheGap, ShortTimeRemunerationFloor-1, false) {
		t.Error("月 87,999 円はロで外れるはず")
	}
}
