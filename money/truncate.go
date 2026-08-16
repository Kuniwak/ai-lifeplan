package money

import "fmt"

const (
	TaxableIncomeUnit Yen = 1000

	IncomeTaxUnit Yen = 100
)

func (y Yen) Truncate(unit Yen) Yen {
	if unit <= 0 {
		panic(fmt.Sprintf("money: Truncate needs a positive unit, got %d", unit))
	}

	rem := y % unit
	if rem < 0 {
		rem += unit
	}
	return y - rem
}

func (y Yen) TruncateTaxableIncome() Yen {
	return y.Truncate(TaxableIncomeUnit)
}

func (y Yen) TruncateIncomeTax() Yen {
	return y.Truncate(IncomeTaxUnit)
}
