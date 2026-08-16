package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/table"
	"pgregory.net/rapid"
)

const (
	theFirstChainYear = 2019
	theLastChainYear  = 2100
)

const noDependants = 0

func TestMoreDeductionsShouldNeverRaiseTheTax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		income := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "合計所得金額"))
		smaller := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "所得控除の合計"))
		more := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "足す控除"))
		credits := money.Yen(rapid.Int64Range(0, 5_000_000).Draw(t, "税額控除"))
		year := date.Year(rapid.IntRange(theFirstChainYear, theLastChainYear).Draw(t, "西暦"))

		with := table.ChainTax(income, smaller+more, credits, noDependants, year)
		without := table.ChainTax(income, smaller, credits, noDependants, year)

		if with.Payable > without.Payable {
			t.Fatalf("控除を %d 増やしたら申告納税額が %d から %d に増えた（合計所得金額 %d、税額控除 %d、%d年）",
				more, without.Payable, with.Payable, income, credits, year)
		}
	})
}

func TestMoreCreditsShouldNeverRaiseTheTax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		income := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "合計所得金額"))
		deductions := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "所得控除の合計"))
		smaller := money.Yen(rapid.Int64Range(0, 5_000_000).Draw(t, "税額控除"))
		more := money.Yen(rapid.Int64Range(0, 5_000_000).Draw(t, "足す税額控除"))
		year := date.Year(rapid.IntRange(theFirstChainYear, theLastChainYear).Draw(t, "西暦"))

		with := table.ChainTax(income, deductions, smaller+more, noDependants, year)
		without := table.ChainTax(income, deductions, smaller, noDependants, year)

		if with.Payable > without.Payable {
			t.Fatalf("税額控除を %d 増やしたら申告納税額が %d から %d に増えた",
				more, without.Payable, with.Payable)
		}
	})
}

func TestTheChainShouldNeverOweMoreThanTheIncome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		income := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "合計所得金額"))
		deductions := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "所得控除の合計"))
		credits := money.Yen(rapid.Int64Range(0, 5_000_000).Draw(t, "税額控除"))
		year := date.Year(rapid.IntRange(theFirstChainYear, theLastChainYear).Draw(t, "西暦"))

		got := table.ChainTax(income, deductions, credits, noDependants, year)

		for _, link := range []struct {
			name string
			at   money.Yen
		}{
			{"課税所得金額", got.Taxable},
			{"算出税額", got.Tax},
			{"差引所得税額", got.BaseTax},
			{"復興特別所得税", got.Surtax},
			{"申告納税額", got.Payable},
		} {
			if link.at < 0 {
				t.Fatalf("%s が %d で負である（合計所得金額 %d、所得控除 %d、税額控除 %d）",
					link.name, link.at, income, deductions, credits)
			}
		}
		if got.Payable > income {
			t.Fatalf("申告納税額 %d が合計所得金額 %d を超えた", got.Payable, income)
		}
	})
}
