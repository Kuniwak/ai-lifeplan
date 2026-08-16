package money

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
)

func (y Yen) ForMonths(months int) Yen {
	if months < 0 || months > date.MonthsAYear {
		panic(fmt.Sprintf("money: ForMonths needs 0..%d months, got %d", date.MonthsAYear, months))
	}
	return y * Yen(months) / date.MonthsAYear
}
