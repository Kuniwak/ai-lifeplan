package law

import "github.com/Kuniwak/lifeplan/money"

const (
	ShortTimeWeeklyHoursFloor = 20

	ShortTimeRemunerationFloor money.Yen = 88_000
)

func EmployeesInsuranceCovers(at Workplace, weeklyHours int, monthlyRemuneration money.Yen, isStudent bool) bool {
	if weeklyHours <= 0 {
		return false
	}

	if !at.shortTime(weeklyHours) {
		return true
	}

	if !at.Specified {
		return false
	}

	excluded := weeklyHours < ShortTimeWeeklyHoursFloor ||
		monthlyRemuneration < ShortTimeRemunerationFloor ||
		isStudent
	return !excluded
}

func EmploymentInsuranceCovers(weeklyHours int, isStudent bool) bool {
	return weeklyHours >= ShortTimeWeeklyHoursFloor && !isStudent
}

func WeeklyHoursOf(at Workplace, written int, monthlyRemuneration money.Yen) int {
	if written <= 0 && monthlyRemuneration > 0 {
		return at.NormalWeeklyHours
	}
	return written
}
