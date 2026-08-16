package law

import (
	"math/big"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

var InvestmentIncomeTaxRate = money.NewRate(20_315, 100_000)

func SellForCash(need, held, basis money.Yen) (sold, tax money.Yen) {
	if held <= 0 || need <= 0 {
		return 0, 0
	}

	gain := held - basis
	if gain <= 0 {
		return min(need, held), 0
	}

	num := new(big.Int).SetInt64(int64(need))
	num.Mul(num, big.NewInt(int64(held)))
	num.Mul(num, big.NewInt(InvestmentIncomeTaxRate.Den()))

	den := new(big.Int).SetInt64(int64(held))
	den.Mul(den, big.NewInt(InvestmentIncomeTaxRate.Den()))
	den.Sub(den, new(big.Int).Mul(big.NewInt(int64(gain)), big.NewInt(InvestmentIncomeTaxRate.Num())))

	num.Add(num, den)
	num.Sub(num, big.NewInt(1))
	sold = min(money.Yen(new(big.Int).Div(num, den).Int64()), held)

	return sold, InvestmentIncomeTaxOn(GainOn(sold, gain, held))
}

func InvestmentIncomeTaxOn(gain money.Yen) money.Yen {
	if gain <= 0 {
		return 0
	}
	return gain.Mul(InvestmentIncomeTaxRate, money.Truncate)
}

func GainOn(sold, gain, held money.Yen) money.Yen {
	return money.ShareOf(sold, gain, held)
}

const TsumitateNISAExemptYears = 20

func TsumitateNISAMaturityOf(boughtIn date.Year) date.Year {
	return boughtIn + TsumitateNISAExemptYears - 1
}

func NISAMaturityBasis(marketValue money.Yen) money.Yen { return marketValue }
