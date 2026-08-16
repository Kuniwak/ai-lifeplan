package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

type Cover string

const (
	NoCover Cover = ""

	EmployeesHealthInsurance Cover = "社会保険"

	NationalHealthInsurance Cover = "国民健康保険"

	LateElderlyHealthCare Cover = "後期高齢者医療制度"
)

func LateElderlyFrom(born date.Date) date.Date {
	return born.Anniversary(LateElderlyAge)
}

func LateElderlyMonthsIn(year date.Year, born date.Date) date.Months {
	return date.MonthsOfYearFromIn(year, LateElderlyFrom(born))
}

type CoverMonths struct {
	Cover  Cover
	Months date.Months
}

func CoverMonthsIn(year date.Year, born date.Date, otherwise Cover) []CoverMonths {
	if otherwise == NoCover {
		return nil
	}

	late := LateElderlyMonthsIn(year, born)
	under := make([]CoverMonths, 0, 2)
	if before := date.WholeYear &^ late; before != date.NoMonths {
		under = append(under, CoverMonths{Cover: otherwise, Months: before})
	}
	if late != date.NoMonths {
		under = append(under, CoverMonths{Cover: LateElderlyHealthCare, Months: late})
	}
	return under
}

func LongestCover(months []CoverMonths) Cover {
	var longest CoverMonths
	for _, under := range months {
		if under.Months.Count() > longest.Months.Count() {
			longest = under
		}
	}
	return longest.Cover
}

func EmployeeCoverIn(year date.Year, born date.Date, at Workplace, weeklyHours int, monthlyRemuneration money.Yen, isStudent bool) Cover {
	return LongestCover(CoverMonthsIn(year, born, EmployeeCoverFrom(at, weeklyHours, monthlyRemuneration, isStudent)))
}

func EmployeeCoverFrom(at Workplace, weeklyHours int, monthlyRemuneration money.Yen, isStudent bool) Cover {
	if EmployeesInsuranceCovers(at, weeklyHours, monthlyRemuneration, isStudent) {
		return EmployeesHealthInsurance
	}
	return NationalHealthInsurance
}
