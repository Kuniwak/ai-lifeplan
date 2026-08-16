package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func incomeTaxOfTheBaseProject(t *testing.T, taxpayer, spouse table.PersonName) relation.Table[table.IncomeTaxRow] {
	t.Helper()

	built, err := table.IncomeTaxTable(incomeTaxInputOfTheBaseProject(t, taxpayer, spouse))
	if err != nil {
		t.Fatalf("table.IncomeTaxTable: %v", err)
	}
	return built
}

func incomeTaxInputOfTheBaseProject(t *testing.T, taxpayer, spouse table.PersonName) table.IncomeTaxInput {
	t.Helper()

	calendar := calendarOfTheBaseProject(t)
	lawFS := os.DirFS("../" + law.LawDirectory)

	disability := law.MustLoadDisabilityDeductions(t, lawFS)
	housing := law.MustLoadHousingLoanCredits(t, lawFS)
	spouseCeiling := law.MustLoadSpouseIncomeCeilings(t, lawFS)

	income := make(map[table.PersonName]relation.Table[money.Yen], 2)
	for person, slot := range map[table.PersonName]tsv.Slot{"夫": input.IncomeHusbandSlot, "妻": input.IncomeWifeSlot} {
		built := incomeOfTheBaseProject(t, person, slot)
		rows := make([]relation.Row[money.Yen], 0, built.Len())
		for _, row := range built.Rows() {
			rows = append(rows, relation.Row[money.Yen]{Year: row.Year, Value: row.Value.TotalIncome})
		}
		income[person] = relation.New(rows)
	}

	social := make([]relation.Row[money.Yen], 0, calendar.Len())
	for _, row := range socialInsuranceTotalOfTheBaseProject(t).Rows() {
		paid := money.Yen(0)
		if taxpayer == "夫" {
			paid = row.Value.Total
		}
		social = append(social, relation.Row[money.Yen]{Year: row.Year, Value: paid})
	}

	expense, err := table.ExpenseInputFrom(tablesOfTheBaseProject(t), calendar)
	if err != nil {
		t.Fatalf("table.ExpenseInputFrom: %v", err)
	}
	loan, err := theLoanOfTheBaseProject.LoanTable(nil, planStart, planEnd, theFloatingOfTheBaseProject(planStart, planEnd))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}
	balance := make([]relation.Row[money.Yen], 0, loan.Len())
	for _, row := range loan.Rows() {
		balance = append(balance, relation.Row[money.Yen]{Year: row.Year, Value: row.Value.Balance})
	}

	built, err := table.ExpenseTable(expense)
	if err != nil {
		t.Fatalf("table.ExpenseTable: %v", err)
	}
	quake := relation.Map(built, func(_ date.Year, r table.ExpenseRow) money.Yen { return r.EarthquakeDeductible })
	medicalPaid, medicalRefunded := medicalColumns(t, calendar.Years())

	movedIn := date.Year(2022)
	in := table.IncomeTaxInput{
		Calendar:        calendar,
		Taxpayer:        taxpayer,
		Spouse:          spouse,
		Income:          income,
		SocialInsurance: relation.New(social),
		Disabilities: map[table.PersonName]table.Disability{
			"妻": {Category: law.OrdinaryDisability, CertifiedIn: 2022},
		},
		HousingLoanBalance:  relation.New(balance),
		DisabilityDeduction: disability,
		HousingLoanCredit:   housing,
		SpouseIncomeCeiling: spouseCeiling,
	}

	if taxpayer == "夫" {
		in.ClaimsDependents = true
		in.MovedIn = &movedIn
		in.LifeInsurancePremium = expense.LifeInsurance
		in.MedicalInsurancePremium = expense.MedicalInsurance
		in.MutualAidContribution = mutualAidOfTheBaseProject(t, calendar.Years())
		in.EarthquakePremium = quake
		in.MedicalPaid, in.MedicalRefunded = medicalPaid, medicalRefunded
	}

	return in
}

func mutualAidOfTheBaseProject(t *testing.T, years []date.Year) relation.Table[money.Yen] {
	t.Helper()

	built, err := table.ReadYenStep(
		tablesOfTheBaseProject(t)[input.MutualAidContributionSlot], input.MutualAidContributionSlot,
		input.MutualAidContributionColumn, years[0], years[len(years)-1])
	if err != nil {
		t.Fatalf("read %s: %v", input.MutualAidContributionColumn, err)
	}
	return built
}

func eventsAsTable(events map[date.Year]money.Yen, years []date.Year) relation.Table[money.Yen] {
	return relation.Over(years, func(y date.Year) money.Yen { return events[y] })
}

func medicalColumns(t *testing.T, years []date.Year) (paid, refunded relation.Table[money.Yen]) {
	t.Helper()

	tables := tablesOfTheBaseProject(t)
	read := func(column tsv.ColumnName) relation.Table[money.Yen] {
		expanded, err := table.ReadYenStep(tables[input.MedicalExpenseSlot], input.MedicalExpenseSlot, column, years[0], years[len(years)-1])
		if err != nil {
			t.Fatalf("read %s: %v", column, err)
		}
		return expanded
	}
	return read(input.MedicalColumn), read(input.MedicalRefundColumn)
}

func TestTheMutualAidDeductionShouldCountAgainstBothTaxes(t *testing.T) {
	const paid money.Yen = 276_000

	total := table.Deductions{MutualAid: paid}.Total()
	if total.IncomeTax != paid {
		t.Errorf("所得税の所得控除: %d, want %d", total.IncomeTax, paid)
	}
	if total.Resident != paid {
		t.Errorf("住民税の所得控除: %d, want %d", total.Resident, paid)
	}
}

func TestNobodyWithoutACertificateShouldBeDisabled(t *testing.T) {
	certified := table.Disabilities{"妻": {CertifiedIn: 2022}}

	cases := map[string]struct {
		Name table.PersonName
		Year date.Year
		Want bool
	}{
		"証明のある人・認定年より前":   {Name: "妻", Year: 2021, Want: false},
		"証明のある人・認定年（境界値）": {Name: "妻", Year: 2022, Want: true},
		"証明のある人・認定年より後":   {Name: "妻", Year: 2030, Want: true},
		"証明の無い人":          {Name: "夫", Year: 2030, Want: false},
		"証明の無い人・はるか前":     {Name: "夫", Year: 1900, Want: false},
		"表そのものに載っていない人":   {Name: "長男", Year: 2030, Want: false},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := certified.AppliesTo(c.Name, c.Year)

			if got != c.Want {
				t.Errorf("AppliesTo(%q, %d) = %v, want %v", c.Name, c.Year, got, c.Want)
			}
		})
	}
}
