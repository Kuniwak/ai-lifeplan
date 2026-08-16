package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func aLoan(name table.LoanName, principal money.Yen, rate money.Rate) table.Loan {
	return table.Loan{
		Name: name, Principal: principal, AnnualRate: rate,
		Years: 35, FixedYears: 35, DrawnIn: 2022, FirstYear: 2023, FirstMonth: 1,
	}
}

func ratesFor(loans []table.Loan, from, to date.Year) map[table.LoanName]relation.Table[money.Rate] {
	real := make(map[table.LoanName]money.Rate, len(loans))
	for _, l := range loans {
		real[l.Name] = money.NewRate(0, 100)
	}
	flat := relation.Constant(relation.Span(from, to), money.NewRate(0, 100))
	return table.NominalRatesByLoan(real, from, to, flat)
}

func TestLoansTableShouldAddTheContractsTogether(t *testing.T) {
	two := []table.Loan{
		aLoan("土地", 18_000_000, money.NewRate(1788, 100_000)),
		aLoan("住宅", 35_000_000, money.NewRate(2108, 100_000)),
	}

	together, err := table.LoansTable(two, nil, 2022, 2060, ratesFor(two, 2022, 2060))
	if err != nil {
		t.Fatalf("LoansTable: %v", err)
	}

	for _, loan := range two {
		one, err := loan.LoanTable(nil, 2022, 2060, noInflation(2022, 2060))
		if err != nil {
			t.Fatalf("LoanTable(%s): %v", loan.Name, err)
		}
		for _, row := range one.Rows() {
			sum, _ := together.At(row.Year)
			if sum.Balance < row.Value.Balance {
				t.Errorf("%d: 合計の残高 %d が %s ひとつの %d を下回っている",
					row.Year, sum.Balance, loan.Name, row.Value.Balance)
			}
		}
	}

	first, _ := together.At(2022)
	if want := money.Yen(53_000_000); first.Balance != want {
		t.Errorf("2022 年末の残高 %d が二本の元本の和 %d と違う", first.Balance, want)
	}
}

func TestLoansTableShouldRefuseAContractWithNoRate(t *testing.T) {
	loans := []table.Loan{aLoan("土地", 18_000_000, money.NewRate(2, 100))}

	if _, err := table.LoansTable(loans, nil, 2022, 2060, map[table.LoanName]relation.Table[money.Rate]{}); err == nil {
		t.Error("変動金利の渡されていない契約を拒んでいない")
	}
}
