package plan

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func assertTheRecordedYearsAreAtTodaysPrices(levels relation.Table[money.Factor], balance actuals.BalanceTable) error {
	last, _, ok := balance.Latest()
	if !ok {
		return nil
	}

	for _, row := range levels.Rows() {
		if row.Year > last {
			continue
		}
		if row.Value.IsOne() {
			continue
		}
		return fmt.Errorf(
			"plan: %d 年の物価が %s である。この年は実績が記録されている年（残高の最終行は %d 年）で、"+
				"世帯が書いた額はその年の物価で書かれている。**観測された消費者物価指数をここに書くと、"+
				"払った額そのものが動く**——2022 年の頭金 8,915,000 円が 9,165,000 円になる。"+
				"data/environment/inflation.tsv の %d 年までは 0%% であること",
			row.Year, row.Value, last, last)
	}
	return nil
}

const pensionTableYear date.Year = 2026

func recordedRemunerationOf(record []actuals.Remuneration, person string) []law.Remuneration {
	out := make([]law.Remuneration, 0, len(record))
	for _, one := range record {
		if one.Person != person || !one.Known {
			continue
		}
		out = append(out, law.Remuneration{Year: one.Year, Month: one.Month, Amount: one.Amount})
	}
	return out
}

func latestRecordedOf(record []actuals.Remuneration, person string) law.Remuneration {
	year, month := actuals.LatestRecorded(record, person)
	return law.Remuneration{Year: year, Month: month}
}

func deferredMonths(calendar relation.Table[table.CalendarRow], person table.PersonName, start date.Year) int {
	bornOn, err := table.BornOnIn(calendar, person)
	if err != nil {
		return 0
	}
	sixtyFive := bornOn.ReachesAge(law.PensionDeductionAge).Year
	return int(start-sixtyFive) * date.MonthsAYear
}

func supplementaryPensionFor(
	in2 *Input, calendar relation.Table[table.CalendarRow],
	person table.PersonName, pay map[table.PersonName]relation.Table[table.Pay],
	pension *table.Pension,
) error {
	if person != Earner {
		return nil
	}

	spouseBorn, err := table.BornOnIn(calendar, Spouse)
	if err != nil {
		return nil
	}
	if err := law.AssertSupplementaryPensionSpecialAddition(spouseBorn); err != nil {
		return err
	}

	months := make(map[table.PersonName]int, 2)
	for _, who := range []table.PersonName{Earner, Spouse} {
		recorded, err := actuals.LatestPensionRecordItem(in2.pensionRecord, string(who), actuals.EmployeePensionMonthsItem)
		if err != nil {
			return err
		}
		if months[who], err = table.EmployeePensionMonths(table.EmployeePensionMonthsInput{
			Recorded:        recorded.Value,
			RecordedThrough: recorded.AsOf,
			Pay:             pay[who],
			Calendar:        calendar,
			Person:          who,
		}); err != nil {
			return err
		}
	}
	if law.SpouseSupplementaryPensionSuspended(months[Spouse]) {
		return nil
	}

	amount, special := in2.statutes.supplementaryPension, in2.statutes.supplementarySpecial
	pension.Supplement = law.SpouseSupplementaryPension(amount, special, months[Earner])

	earnerBorn, err := table.BornOnIn(calendar, person)
	if err != nil {
		return err
	}
	payable := date.FirstOfMonth(earnerBorn.ReachesAge(law.PensionDeductionAge))
	if january := (date.Date{Year: pension.StartYear, Month: 1, Day: 1}); payable.Before(january) {
		payable = january
	}
	from := law.SpouseSupplementaryPensionFrom(payable)
	through := law.SpouseSupplementaryPensionThrough(spouseBorn)
	if !from.Before(through) {
		pension.Supplement = 0
		return nil
	}
	pension.SupplementFrom, pension.SupplementThrough = from, through
	return nil
}

func build(in2 *Input) (*Plan, error) {
	plain, err := buildWith(in2, table.LastResort{})
	if err != nil {
		return nil, err
	}

	measures, err := table.Measures(in2.tables[input.LastResortSlot])
	if err != nil {
		return nil, err
	}
	short := firstShortfallYear(plain)
	if len(measures) == 0 || short == 0 {
		return plain, nil
	}

	best := plain
	for _, name := range []table.MeasureName{table.ReverseMortgage, table.SellAndRent} {
		measure, ok := measures[name]
		if !ok {
			continue
		}
		resort, err := plain.lastResortOf(in2.tables, measure, short)
		if err != nil {
			return nil, err
		}
		if resort.From == 0 {
			continue
		}
		candidate, err := buildWith(in2, resort)
		if err != nil {
			return nil, err
		}
		if betterThan(candidate, best) {
			best = candidate
		}
	}
	return best, nil
}

func firstShortfallYear(p *Plan) date.Year {
	for _, row := range p.Assets.Rows() {
		if row.Value.Shortfall > 0 {
			return row.Year
		}
	}
	return 0
}

func betterThan(a, b *Plan) bool {
	aRuin, aShort := ruinOf(a)
	bRuin, bShort := ruinOf(b)
	switch {
	case (aRuin == 0) != (bRuin == 0):
		return aRuin == 0
	case aRuin != bRuin:
		return aRuin > bRuin
	case aShort != bShort:
		return aShort < bShort
	}
	return finalAssets(a) > finalAssets(b)
}

func ruinOf(p *Plan) (date.Year, money.Yen) {
	var first date.Year
	var total money.Yen
	for _, row := range p.Assets.Rows() {
		if row.Value.Shortfall <= 0 {
			continue
		}
		if first == 0 {
			first = row.Year
		}
		if level, ok := p.PriceLevels.At(row.Year); ok {
			total += level.Deflate(row.Value.Shortfall)
		}
	}
	return first, total
}

func finalAssets(p *Plan) money.Yen {
	rows := p.Assets.Rows()
	if len(rows) == 0 {
		return 0
	}
	return rows[len(rows)-1].Value.Total
}

func buildWith(in2 *Input, resort table.LastResort) (*Plan, error) {
	tables, s, balance, cash := in2.tables, in2.statutes, in2.balance, in2.cash

	p := &Plan{
		Income:      make(map[table.PersonName]relation.Table[table.IncomeRow], 2),
		IncomeTax:   make(map[table.PersonName]relation.Table[table.IncomeTaxRow], 2),
		ResidentTax: make(map[table.PersonName]relation.Table[table.ResidentTaxRow], 2),
	}

	calendarInput, err := table.CalendarInputFrom(tables)
	if err != nil {
		return nil, err
	}
	if p.Calendar, err = table.Calendar(calendarInput); err != nil {
		return nil, err
	}
	years := p.Calendar.Years()
	from, to := years[0], years[len(years)-1]

	if p.PriceLevels, err = table.PriceLevelsFrom(tables, from, to); err != nil {
		return nil, err
	}
	if p.levelsByItem, err = table.PriceLevelsByItemFrom(tables, from, to); err != nil {
		return nil, err
	}
	if err := assertTheRecordedYearsAreAtTodaysPrices(p.PriceLevels, balance); err != nil {
		return nil, err
	}

	if p.mutualAid, err = readInflatedYenStep(tables, p.levelsByItem,
		input.MutualAidContributionSlot, input.MutualAidContributionColumn,
		input.MutualAidItem, from, to); err != nil {
		return nil, err
	}

	loan, boughtIn, err := readLoan(tables, from, to, balance)
	if err != nil {
		return nil, err
	}
	p.Loan = loan

	payTables := map[table.PersonName]tsv.Slot{Earner: input.IncomeHusbandSlot, Spouse: input.IncomeWifeSlot}

	nominalPay := make(map[table.PersonName]relation.Table[table.Pay], len(payTables))

	incomeInputs := make(map[table.PersonName]table.IncomeInput, len(payTables))
	for person, slot := range payTables {
		in, err := table.IncomeInputFor(tables, p.Calendar, person, slot)
		if err != nil {
			return nil, err
		}
		in.ChildcareLeave = s.childcareLeave

		if nominalPay[person], err = in.NominalPay(); err != nil {
			return nil, err
		}
		incomeInputs[person] = in
	}

	for person := range payTables {
		in := incomeInputs[person]

		deferred := deferredMonths(p.Calendar, person, in.Pension.StartYear)

		proportional, err := table.EarningsRelatedPension(table.EarningsRelatedPensionInput{
			Recorded:        recordedRemunerationOf(in2.remuneration, string(person)),
			RecordedThrough: latestRecordedOf(in2.remuneration, string(person)),
			Pay:             nominalPay[person],
			Grades:          s.pensionGrades,
			Rates:           s.pensionRevaluation,
			Published:       pensionTableYear,
			RevaluedFrom:    pensionTableYear,
			DeferredMonths:  deferred,
			Calendar:        p.Calendar,
			Person:          person,
		})
		if err != nil {
			return nil, err
		}
		in.Pension.Proportional = proportional

		recorded, err := actuals.LatestPensionRecordItem(
			in2.pensionRecord, string(person), actuals.PaidNationalPensionMonthsItem)
		if err != nil {
			return nil, err
		}
		paid, err := table.NationalPensionPaidMonths(table.NationalPensionPaidMonthsInput{
			PaidInTheRecord: recorded.Value,
			RecordedThrough: recorded.AsOf,
			Monthly:         in2.remuneration,
			Calendar:        p.Calendar,
			Person:          person,
		})
		if err != nil {
			return nil, err
		}
		basic, err := table.BasicPension(table.BasicPensionInput{
			Full:           s.basicPensionFull.Amount(pensionTableYear),
			PaidMonths:     paid,
			DeferredMonths: deferred,
		})
		if err != nil {
			return nil, err
		}
		in.Pension.Basic = basic

		if err := supplementaryPensionFor(in2, p.Calendar, person, nominalPay, &in.Pension); err != nil {
			return nil, err
		}

		level, err := table.PensionLevelAt(tables, in.Pension.StartYear)
		if err != nil {
			return nil, err
		}
		in.Pension = in.Pension.AtLevel(level)

		if p.Income[person], err = table.IncomeTable(in); err != nil {
			return nil, err
		}
	}

	growth, err := table.CostGrowthFrom(tables, from, to)
	if err != nil {
		return nil, err
	}

	for _, c := range []struct {
		what        string
		assumption  law.CostGrowthCurve
		lastWritten law.LastWrittenYear
	}{
		{"健康保険料率", growth.Medical, s.socialRates.Health.LastWrittenYear},
		{"後期高齢者医療", growth.Medical, s.kouki.LastWrittenYear},
		{"国民健康保険", growth.Medical, s.kokuho.LastWrittenYear},

		{"介護保険料率", growth.Care, s.socialRates.NursingCare.LastWrittenYear},
		{"介護保険料（第1号被保険者）", growth.CarePremium, s.nursingCare.LastWrittenYear},
	} {
		last, ok := c.lastWritten()
		if !ok {
			continue
		}
		if err := c.assumption.AssertCovers(c.what, last, to); err != nil {
			return nil, err
		}
	}

	for _, c := range []struct {
		what         string
		firstWritten law.FirstWrittenYear
	}{
		{"健康保険料率", s.socialRates.Health.FirstWrittenYear},
		{"介護保険料率", s.socialRates.NursingCare.FirstWrittenYear},
		{"雇用保険料率", s.employment.FirstWrittenYear},
	} {
		if err := law.AssertRecordReaches(c.what, c.firstWritten, from); err != nil {
			return nil, err
		}
	}

	for _, floor := range law.RecordFloors() {
		if err := law.AssertRecordReaches(floor.What, floor.FirstWritten, from); err != nil {
			return nil, err
		}
	}

	for _, c := range []struct {
		what         string
		age          int
		firstWritten law.FirstWrittenYear
	}{
		{"介護保険料（第1号被保険者）", law.NursingCareFirstCategoryAge, s.nursingCare.LastWrittenYear},
		{"後期高齢者医療", law.LateElderlyAge, s.kouki.FirstWrittenYear},
		{"国民年金保険料", law.NationalPensionFromAge, s.nationalPension.FirstWrittenYear},
	} {
		asked, ever := table.FirstYearAnybodyReaches(p.Calendar, c.age)
		if !ever {
			continue
		}
		if err := law.AssertRecordReaches(c.what, c.firstWritten, asked); err != nil {
			return nil, err
		}
	}

	employees := make(map[table.PersonName]relation.Table[table.SocialInsuranceRow], len(payTables))
	for person, pay := range nominalPay {
		built, err := table.SocialInsuranceTable(table.SocialInsuranceInput{
			Pay: pay, Calendar: p.Calendar, Person: person,
			HealthGrades: s.healthGrades, PensionGrades: s.pensionGrades,
			Rates: s.socialRates.WithGrowth(growth), EmploymentInsurance: s.employment,
			Recorded: actuals.PremiumsRecordedByYear(in2.payslips, string(person)),
		})
		if err != nil {
			return nil, err
		}
		employees[person] = built
	}

	totalIncome := relation.MapEach(p.Income,
		func(_ date.Year, r table.IncomeRow) money.Yen { return r.TotalIncome })

	disabilities, err := table.DisabilitiesFrom(tables)
	if err != nil {
		return nil, err
	}
	taxed := make(map[table.PersonName]relation.Table[law.ResidentTaxLiability], 2)
	for _, person := range []table.PersonName{Earner, Spouse} {
		spouse := Earner
		if person == Earner {
			spouse = Spouse
		}
		taxed[person], err = table.ResidentTaxLiabilityTable(table.ResidentTaxLiabilityInput{
			Calendar: p.Calendar,
			Dependents: table.DependentsInput{
				Taxpayer:            person,
				Spouse:              spouse,
				Income:              totalIncome,
				SpouseIncomeCeiling: s.spouseCeiling,
				ClaimsDependents:    person == Earner,
			},
			Disabilities: disabilities,
			Levies:       s.residentTax,
		})
		if err != nil {
			return nil, err
		}
	}

	members, err := table.HouseholdMembersOf(p.Calendar, p.Income, taxed)
	if err != nil {
		return nil, err
	}

	household, err := table.HouseholdInsuranceTable(table.HouseholdInsuranceInput{
		Calendar:        p.Calendar,
		Members:         members,
		EmployeeCovers:  coversOf(employees),
		Disabilities:    disabilities,
		Kokuho:          s.kokuho.WithGrowth(growth),
		Kouki:           s.kouki.WithGrowth(growth),
		NationalPension: s.nationalPension,
		NursingCare:     s.nursingCare.WithGrowth(growth),
	})
	if err != nil {
		return nil, err
	}

	p.HouseholdInsurance = household

	if p.SocialInsurance, err = table.SocialInsuranceTotalTable(table.SocialInsuranceTotalInput{
		Employees: employees,
		Household: household,
	}); err != nil {
		return nil, err
	}

	if p.ChildAllowance, err = table.ChildAllowanceTable(table.ChildAllowanceInput{
		Calendar:           p.Calendar,
		HigherEarnerIncome: totalIncome[Earner],
		Table:              s.childAllowance,
	}); err != nil {
		return nil, err
	}

	expenseInput, err := table.ExpenseInputFrom(tables, p.Calendar)
	if err != nil {
		return nil, err
	}
	expenseInput.Loan = p.Loan

	if err := p.buildSpendingAndTaxes(tables, s, expenseInput, taxed, employees, boughtIn, from, to); err != nil {
		return nil, err
	}

	if p.Timeline, err = table.TimelineTable(table.TimelineInput{
		Income:          p.Income,
		ChildAllowance:  p.ChildAllowance,
		Expense:         p.Expense,
		SocialInsurance: p.SocialInsurance,
		Tax:             p.Tax,
	}); err != nil {
		return nil, err
	}

	p.LastResort = resort
	if err := p.buildAssets(tables, s, balance, from, to, resort); err != nil {
		return nil, err
	}
	return p, p.buildOutturn(tables, balance, cash)
}

func readLoan(tables map[tsv.Slot]*tsv.Table, from, to date.Year, balance actuals.BalanceTable) (relation.Table[table.LoanYear], *date.Year, error) {
	var empty relation.Table[table.LoanYear]

	loans, floatingReal, settled, boughtIn, err := table.LoansFrom(tables)
	if err != nil {
		return empty, nil, err
	}

	if boughtIn == nil || len(loans) == 0 {
		return relation.Constant(relation.Span(from, to), table.LoanYear{}), nil, nil
	}

	if settled != nil {
		if startsAfter, _, ok := balance.Latest(); ok && *settled <= startsAfter {
			return empty, nil, fmt.Errorf(
				"plan: 一括返済の年 %d は実績のある年である。実績は %d 年までで、その年までの残高は記録が決めるので、返済は消えてしまう",
				*settled, startsAfter)
		}
	}

	prices, err := table.InflationRatesFrom(tables, from, to)
	if err != nil {
		return empty, nil, err
	}
	built, err := table.LoansTable(loans, settled, from, to,
		table.NominalRatesByLoan(floatingReal, from, to, prices))
	return built, boughtIn, err
}

func coverOf(employee relation.Table[table.SocialInsuranceRow]) relation.Table[law.Cover] {
	return relation.Map(employee, func(_ date.Year, r table.SocialInsuranceRow) law.Cover {
		return r.Cover
	})
}

func coversOf(employees map[table.PersonName]relation.Table[table.SocialInsuranceRow]) map[table.PersonName]relation.Table[law.Cover] {
	covers := make(map[table.PersonName]relation.Table[law.Cover], len(employees))
	for person, built := range employees {
		covers[person] = coverOf(built)
	}
	return covers
}

const MedicalProjectionRounds = 30

func readInflatedYenStep(
	tables map[tsv.Slot]*tsv.Table,
	levels map[input.PricedItem]relation.Table[money.Factor],
	slot tsv.Slot, column tsv.ColumnName, item input.PricedItem,
	from, to date.Year,
) (relation.Table[money.Yen], error) {
	written, err := table.ReadYenStep(tables[slot], slot, column, from, to)
	if err != nil {
		return relation.Table[money.Yen]{}, err
	}
	return table.InflateItem(levels, item, written)
}

func (p *Plan) buildSpendingAndTaxes(
	tables map[tsv.Slot]*tsv.Table, s statutes, expenseInput table.ExpenseInput,
	taxed map[table.PersonName]relation.Table[law.ResidentTaxLiability],
	employees map[table.PersonName]relation.Table[table.SocialInsuranceRow],
	boughtIn *date.Year, from, to date.Year,
) error {
	curve, err := table.AgeCurveFrom(tables)
	if err != nil {
		return err
	}
	recordedTo, err := table.LastRecordedMedicalYear(tables)
	if err != nil {
		return err
	}
	recorded := expenseInput.MedicalPaid

	for round := 0; ; round++ {
		if p.Expense, err = table.ExpenseTable(expenseInput); err != nil {
			return err
		}
		if err := p.buildTaxes(tables, s, expenseInput, taxed, employees, boughtIn, from, to); err != nil {
			return err
		}

		projected, err := table.MedicalProjection(table.MedicalProjectionInput{
			Calendar:   p.Calendar,
			Recorded:   recorded,
			RecordedTo: recordedTo,
			Curve:      curve,
			Copay:      table.MedicalCopayFrom(p.ResidentTax, p.Income),
			Projected:  adults(p.Calendar),
		})
		if err != nil {
			return err
		}
		if relation.SameEveryYear(projected, expenseInput.MedicalPaid) {
			return nil
		}
		if round+1 >= MedicalProjectionRounds {
			return fmt.Errorf(
				"plan: 医療費の見込みと税が %d 回まわしても一致しない。窓口負担割合が年ごとに揺れ続けている",
				MedicalProjectionRounds)
		}
		expenseInput.MedicalPaid = projected
	}
}

func (p *Plan) buildTaxes(
	tables map[tsv.Slot]*tsv.Table, s statutes, expense table.ExpenseInput,
	taxed map[table.PersonName]relation.Table[law.ResidentTaxLiability],
	employees map[table.PersonName]relation.Table[table.SocialInsuranceRow],
	boughtIn *date.Year, from, to date.Year,
) error {

	income := relation.MapEach(p.Income,
		func(_ date.Year, r table.IncomeRow) money.Yen { return r.TotalIncome })

	deductible, err := table.SocialInsuranceDeductions(p.HouseholdInsurance, employees, Earner)
	if err != nil {
		return err
	}

	balance := make([]relation.Row[money.Yen], 0, p.Loan.Len())
	for _, row := range p.Loan.Rows() {
		balance = append(balance, relation.Row[money.Yen]{Year: row.Year, Value: row.Value.Balance})
	}

	quake := spendingOf(p.Expense, func(r table.ExpenseRow) money.Yen { return r.EarthquakeDeductible })
	life := spendingOf(p.Expense, func(r table.ExpenseRow) money.Yen { return r.Life })
	medicalCover := spendingOf(p.Expense, func(r table.ExpenseRow) money.Yen { return r.MedicalCover })
	medicalPaid := spendingOf(p.Expense, func(r table.ExpenseRow) money.Yen { return r.MedicalPaid })
	medicalRefunded := spendingOf(p.Expense, func(r table.ExpenseRow) money.Yen { return r.MedicalRefunded })

	disabilities, err := table.DisabilitiesFrom(tables)
	if err != nil {
		return err
	}

	for _, person := range []table.PersonName{Earner, Spouse} {
		in := table.IncomeTaxInput{
			Calendar:            p.Calendar,
			Taxpayer:            person,
			Income:              income,
			SocialInsurance:     deductible[person],
			Disabilities:        disabilities,
			HousingLoanBalance:  relation.New(balance),
			DisabilityDeduction: s.disability,
			HousingLoanCredit:   s.housingLoan,
			SpouseIncomeCeiling: s.spouseCeiling,
		}

		if person == Earner {
			in.Spouse = Spouse
			in.ClaimsDependents = true
			in.LifeInsurancePremium = life
			in.MedicalInsurancePremium = medicalCover
			in.MutualAidContribution = p.mutualAid
			in.EarthquakePremium = quake
			in.MedicalPaid, in.MedicalRefunded = medicalPaid, medicalRefunded
			in.MovedIn = boughtIn
		}

		built, err := table.IncomeTaxTable(in)
		if err != nil {
			return err
		}
		p.IncomeTax[person] = built
	}

	for _, person := range []table.PersonName{Earner, Spouse} {
		liable, ok := taxed[person]
		if !ok {
			return fmt.Errorf("plan: %s の非課税判定が組み立てられていない", person)
		}

		built, err := table.ResidentTaxTable(table.ResidentTaxInput{
			Calendar:  p.Calendar,
			IncomeTax: p.IncomeTax[person],
			Tables:    s.residentTax,
			Liable:    liable,
		})
		if err != nil {
			return err
		}
		p.ResidentTax[person] = built
	}

	property, err := table.PropertyTaxFrom(tables, p.Calendar, s.depreciation, s.propertyTax,
		p.levelsByItem[input.MaintenanceItem])
	if err != nil {
		return err
	}
	p.PropertyTax = property

	p.Tax, err = table.TaxTotalTable(table.TaxTotalInput{
		IncomeTax: p.IncomeTax, ResidentTax: p.ResidentTax, Property: p.PropertyTax,
	})
	return err
}

func (p *Plan) buildAssets(tables map[tsv.Slot]*tsv.Table, s statutes, balance actuals.BalanceTable, from, to date.Year, resort table.LastResort) error {
	startsAfter, opening, ok := balance.Latest()
	if !ok {
		return fmt.Errorf("plan: the actuals carry no year to start from")
	}

	recorded := make(map[date.Year]table.Balance, len(balance.Years()))
	for _, y := range balance.Years() {
		held, _ := balance.At(y)
		recorded[y] = held
	}

	contribution, err := readInflatedYenStep(tables, p.levelsByItem,
		input.InvestmentSlot, "積立額[円/月]", input.ContributionItem, from, to)
	if err != nil {
		return err
	}
	floor, err := readInflatedYenStep(tables, p.levelsByItem,
		input.InvestmentSlot, "貯蓄維持目標[円]", input.CashFloorItem, from, to)
	if err != nil {
		return err
	}
	allowance, err := table.ReadYenStep(tables[input.InvestmentSlot], input.InvestmentSlot, "NISA生涯投資枠[円]", from, to)
	if err != nil {
		return err
	}
	sellNISAFirst, err := table.SellNISAFirstFrom(tables)
	if err != nil {
		return err
	}

	realReturn, err := table.ReturnsFrom(tables, from, to)
	if err != nil {
		return err
	}
	prices, err := table.InflationRatesFrom(tables, from, to)
	if err != nil {
		return err
	}
	rates := table.NominalReturns(realReturn, prices)
	crash, err := table.CrashesFrom(tables)
	if err != nil {
		return err
	}

	receivedIn, serviceYears := table.PensionReceipt(p.Calendar, Earner, p.mutualAid)

	var residentTax law.ResidentRates
	if receivedIn != nil {
		if row, ok := p.Calendar.At(*receivedIn); ok {
			if tables, known := s.residentTax.TablesFor(row.Municipality); known {
				residentTax, _ = tables.Rates().At(*receivedIn)
			}
		}
	}

	p.Assets, err = table.AssetsTable(table.AssetsInput{
		LastResort:          resort,
		Timeline:            p.Timeline,
		NISAAllowance:       allowance,
		SellNISAFirst:       sellNISAFirst,
		MaturityOfOldNISA:   law.TsumitateNISAMaturityOf(actuals.OldNISABoughtIn),
		Opening:             opening,
		StartsAfter:         startsAfter,
		Actual:              recorded,
		ContributionMonthly: contribution,
		CashFloor:           floor,
		Return:              rates,
		Crash:               crash,
		MutualAid:           p.mutualAid,
		PensionReceivedIn:   receivedIn,
		PensionServiceYears: serviceYears,
		ResidentTax:         residentTax,
	})
	return err
}

func (p *Plan) buildOutturn(tables map[tsv.Slot]*tsv.Table, balance actuals.BalanceTable, cash relation.Table[actuals.CashTakeHomeRow]) error {
	read, ok := tables[input.CashflowSlot]
	if !ok {
		return fmt.Errorf("plan: nothing fills the slot %q, and the outturn has nothing to measure spending against", input.CashflowSlot)
	}

	written, ok := tables[input.AdjustmentsSlot]
	if !ok {
		return fmt.Errorf("plan: nothing fills the slot %q, and the outturn has nothing to add the hand-written spending to", input.AdjustmentsSlot)
	}
	adjustments, err := actuals.ReadAdjustments(written)
	if err != nil {
		return err
	}
	merged, err := actuals.MergeCashflow([]*tsv.Table{read, adjustments})
	if err != nil {
		return err
	}

	spent, err := actuals.YearlyCashflow(merged)
	if err != nil {
		return err
	}

	takeHome := make([]relation.Row[actuals.TakeHome], 0, p.Timeline.Len())
	for _, row := range p.Timeline.Rows() {
		value := actuals.TakeHome{Value: row.Value.TakeHome(), Basis: actuals.AccrualBasis}
		if measured, ok := cash.At(row.Year); ok && measured.Months >= actuals.MonthsInAYear {
			value = actuals.TakeHome{Value: measured.TakeHome, Basis: actuals.CashBasis}
		}
		takeHome = append(takeHome, relation.Row[actuals.TakeHome]{Year: row.Year, Value: value})
	}

	p.Outturn, p.Uncompared, err = actuals.Outturns(balance, relation.New(takeHome), spent)
	return err
}

func spendingOf(expense relation.Table[table.ExpenseRow], of func(table.ExpenseRow) money.Yen) relation.Table[money.Yen] {
	rows := make([]relation.Row[money.Yen], 0, expense.Len())
	for _, row := range expense.Rows() {
		rows = append(rows, relation.Row[money.Yen]{Year: row.Year, Value: of(row.Value)})
	}
	return relation.New(rows)
}

func adults(calendar relation.Table[table.CalendarRow]) []table.PersonName {
	seen := make(map[table.PersonName]bool)
	var names []table.PersonName
	for _, row := range calendar.Rows() {
		for _, person := range row.Value.Ages {
			if person.IsChild() || seen[person.Name] {
				continue
			}
			seen[person.Name] = true
			names = append(names, person.Name)
		}
	}
	return names
}

func (p *Plan) lastResortOf(tables map[tsv.Slot]*tsv.Table, measure table.Measure, short date.Year) (table.LastResort, error) {
	born, err := table.BornOnIn(p.Calendar, Earner)
	if err != nil {
		return table.LastResort{}, err
	}

	cleared := loanClearedIn(p.Loan)
	from := measure.EarliestYear(born, cleared)
	if from < short {
		from = short
	}
	if _, ok := p.Assets.At(from); !ok {
		return table.LastResort{}, nil
	}

	collateral, err := landValue(tables)
	if err != nil {
		return table.LastResort{}, err
	}
	if collateral <= 0 {
		return table.LastResort{}, nil
	}

	resort := table.LastResort{
		Measure:    measure,
		From:       from,
		Collateral: collateral,
		Proceeds:   measure.Proceeds(collateral),
	}
	if !measure.GivesUpHome {
		return resort, nil
	}

	rent := make([]relation.Row[money.Yen], 0, p.Assets.Len())
	owning := make([]relation.Row[money.Yen], 0, p.Assets.Len())
	for _, row := range p.Assets.Rows() {
		level, ok := p.PriceLevels.At(row.Year)
		if !ok {
			return table.LastResort{}, fmt.Errorf("plan.lastResortOf: %d 年の物価が無い", row.Year)
		}
		rent = append(rent, relation.Row[money.Yen]{
			Year:  row.Year,
			Value: level.Apply(measure.RentMonthly * date.MonthsAYear),
		})

		var stops money.Yen
		if tax, ok := p.PropertyTax.At(row.Year); ok {
			stops += tax.Total
		}
		if spend, ok := p.Expense.At(row.Year); ok {
			stops += spend.Maintenance
		}
		owning = append(owning, relation.Row[money.Yen]{Year: row.Year, Value: stops})
	}
	resort.Rent, resort.Owning = relation.New(rent), relation.New(owning)
	return resort, nil
}

func loanClearedIn(loan relation.Table[table.LoanYear]) date.Year {
	var owed bool
	for _, row := range loan.Rows() {
		switch {
		case row.Value.Balance > 0:
			owed = true
		case owed:
			return row.Year
		}
	}
	rows := loan.Rows()
	if len(rows) == 0 {
		return 0
	}
	return rows[len(rows)-1].Year + 1
}

func landValue(tables map[tsv.Slot]*tsv.Table) (money.Yen, error) {
	t, ok := tables[input.PropertyAssessmentSlot]
	if !ok || t == nil || len(t.Rows) == 0 {
		return 0, nil
	}
	r, err := tsv.NewReader(t, input.PropertyAssessmentSlot, input.LandValueColumn)
	if err != nil {
		return 0, err
	}
	return r.Yen(0, input.LandValueColumn)
}
