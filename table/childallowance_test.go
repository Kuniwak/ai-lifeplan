package table_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func childAllowanceOfTheBaseProject(t *testing.T) relation.Table[table.ChildAllowanceRow] {
	t.Helper()

	limits := law.MustLoadChildAllowanceLimits(t, os.DirFS("../"+law.LawDirectory))

	husband := incomeOfTheBaseProject(t, "夫", "income_husband")
	income := make([]relation.Row[money.Yen], 0, husband.Len())
	for _, row := range husband.Rows() {
		income = append(income, relation.Row[money.Yen]{Year: row.Year, Value: row.Value.TotalIncome})
	}

	built, err := table.ChildAllowanceTable(table.ChildAllowanceInput{
		Calendar:           calendarOfTheBaseProject(t),
		HigherEarnerIncome: relation.New(income),
		Table:              limits,
	})
	if err != nil {
		t.Fatalf("table.ChildAllowanceTable: %v", err)
	}
	return built
}

func TestTheLimitsTableShouldExpireTheYearBeforeTheReform(t *testing.T) {
	read := law.MustLoadTable(t, os.DirFS("../"+law.LawDirectory), law.ChildAllowanceLimitsTableName)

	end := columnIndex(t, read, "適用終了年")
	for i, fields := range read.Rows {
		if got := fields[end]; got != fmt.Sprint(law.ChildAllowanceReformedFrom-1) {
			t.Errorf("row %d: 適用終了年 = %s だが、拡充は %d 年から当たる",
				i+1, got, law.ChildAllowanceReformedFrom)
		}
	}
}

func TestTheChildAllowanceThresholdsShouldRiseWithTheNumberOfDependents(t *testing.T) {
	limits := law.MustLoadChildAllowanceLimits(t, os.DirFS("../"+law.LawDirectory))

	const beforeTheReform = law.ChildAllowanceReformedFrom - 1
	one, ok := limits.Limits(beforeTheReform, 1)
	if !ok {
		t.Fatalf("%d 年には限度額があるはずだ", beforeTheReform)
	}
	two, _ := limits.Limits(beforeTheReform, 2)

	if !(two.IncomeLimit > one.IncomeLimit && two.IncomeCeiling > one.IncomeCeiling) {
		t.Errorf("two dependents give %d/%d, no more than the %d/%d of one",
			two.IncomeLimit, two.IncomeCeiling, one.IncomeLimit, one.IncomeCeiling)
	}
}

const childAllowanceLimitsFrom date.Year = 2022
