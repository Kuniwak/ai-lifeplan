package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func expenseOfTheBaseProject(t *testing.T) relation.Table[table.ExpenseRow] {
	t.Helper()

	in, err := table.ExpenseInputFrom(tablesOfTheBaseProject(t), calendarOfTheBaseProject(t))
	if err != nil {
		t.Fatalf("table.ExpenseInputFrom: %v", err)
	}
	loan, err := theLoanOfTheBaseProject.LoanTable(nil, planStart, planEnd, theFloatingOfTheBaseProject(planStart, planEnd))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}
	in.Loan = loan

	built, err := table.ExpenseTable(in)
	if err != nil {
		t.Fatalf("table.ExpenseTable: %v", err)
	}
	return built
}

func TestTheTotalSpendingShouldBeTheSumOfItsFourParts(t *testing.T) {
	for _, row := range expenseOfTheBaseProject(t).Rows() {
		v := row.Value
		if want := v.Living + v.Education + v.Insurance + v.Housing; v.Total != want {
			t.Errorf("%d: 総支出 = %d, the four parts come to %d", row.Year, v.Total, want)
		}
		if want := v.CoupleLiving + v.ChildLiving + v.Medical + v.Allowance + v.Extraordinary; v.Living != want {
			t.Errorf("%d: 生活費合計 = %d, its parts come to %d", row.Year, v.Living, want)
		}
		if want := v.Rent + v.Deposit + v.LoanPaid + v.Maintenance; v.Housing != want {
			t.Errorf("%d: 住宅合計 = %d, its parts come to %d", row.Year, v.Housing, want)
		}
		if v.Total < 0 {
			t.Errorf("%d: 総支出 = %d; spending is never negative", row.Year, v.Total)
		}
	}
}

func TestRecurringShouldExcludeTheDeposit(t *testing.T) {
	for _, row := range expenseOfTheBaseProject(t).Rows() {
		v := row.Value
		if want := v.Total - v.Deposit; v.Recurring != want {
			t.Errorf("%d: 経常支出 = %d, want 総支出 − 頭金 = %d", row.Year, v.Recurring, want)
		}
	}
}

func TestTheHouseholdShouldNotPayRentAndAMortgageInTheSameYear(t *testing.T) {
	for _, row := range expenseOfTheBaseProject(t).Rows() {
		if row.Value.Rent > 0 && row.Value.LoanPaid > 0 {
			t.Errorf("%d: rent %d and mortgage %d in the same year", row.Year, row.Value.Rent, row.Value.LoanPaid)
		}
	}
}

const theFixedPeriodEnds date.Year = 2042

var theYearsTheLivingCostPartsCompany = map[date.Year]bool{}

func personPremium(split []table.PersonPremium, name table.PersonName) money.Yen {
	for _, p := range split {
		if p.Name == name {
			return p.Premium
		}
	}
	return 0
}
