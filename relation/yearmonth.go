package relation

import "github.com/Kuniwak/lifeplan/date"

func MonthsSince(year date.Year, month int) int {
	return int(year)*date.MonthsAYear + month - 1
}

func YearMonthOf(months int) (date.Year, int) {
	return date.Year(months / date.MonthsAYear), months%date.MonthsAYear + 1
}
