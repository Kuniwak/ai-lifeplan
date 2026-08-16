package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type SocialInsuranceRow struct {
	Cover law.Cover

	HealthStandard, PensionStandard money.Yen

	NursingCareInsured bool

	ExemptMonths int

	MonthsCharged int

	HealthOnPay, HealthOnBonus, Health            money.Yen
	NursingCareOnPay, NursingCareOnBonus, Nursing money.Yen
	PensionOnPay, PensionOnBonus, Pension         money.Yen

	EmploymentInsurance money.Yen

	Recorded bool
}

func (r SocialInsuranceRow) Total() money.Yen {
	return r.Health + r.Nursing + r.Pension + r.EmploymentInsurance
}

type SocialInsuranceInput struct {
	Pay relation.Table[Pay]

	Calendar relation.Table[CalendarRow]
	Person   PersonName

	HealthGrades, PensionGrades law.StandardRemunerationTable

	Rates law.SocialInsuranceRates

	EmploymentInsurance law.EmploymentInsuranceTable

	Recorded relation.Table[actuals.PremiumsDeducted]
}

func EmployeeCoverIn(y date.Year, bornOn date.Date, pay Pay) law.Cover {
	const notAStudent = false

	hours := law.WeeklyHoursOf(pay.Workplace, pay.WeeklyHours, pay.Monthly())
	return law.EmployeeCoverIn(y, bornOn, pay.Workplace, hours, pay.Monthly(), notAStudent)
}

func SocialInsuranceTable(in SocialInsuranceInput) (relation.Table[SocialInsuranceRow], error) {
	var empty relation.Table[SocialInsuranceRow]

	bornOn, err := BornOnIn(in.Calendar, in.Person)
	if err != nil {
		return empty, fmt.Errorf("table.SocialInsuranceTable: %w", err)
	}

	const notAStudent = false

	years := in.Pay.Years()
	rows := make([]relation.Row[SocialInsuranceRow], 0, len(years))

	for _, y := range years {
		pay, _ := in.Pay.At(y)
		hours := law.WeeklyHoursOf(pay.Workplace, pay.WeeklyHours, pay.Monthly())

		row := SocialInsuranceRow{Cover: EmployeeCoverIn(y, bornOn, pay)}
		year := y

		if err := assertNoSplitCoverWithPay(y, bornOn, pay); err != nil {
			return empty, fmt.Errorf("table.SocialInsuranceTable: %w", err)
		}

		if law.EmploymentInsuranceCovers(hours, notAStudent) {
			row.EmploymentInsurance = in.EmploymentInsurance.Premium(pay.Salary, year)
		}

		if row.Cover != law.EmployeesHealthInsurance {
			if row, err = readRecorded(row, in.Recorded, y); err != nil {
				return empty, fmt.Errorf("table.SocialInsuranceTable: %w", err)
			}
			rows = append(rows, relation.Row[SocialInsuranceRow]{Year: y, Value: row})
			continue
		}
		nursingMonths := law.NursingCareSecondCategoryMonthsIn(year, bornOn)
		row.NursingCareInsured = nursingMonths != date.NoMonths

		exempt := pay.ExemptMonths
		row.ExemptMonths = exempt.Count()
		row.MonthsCharged = date.MonthsAYear - row.ExemptMonths

		charged := date.WholeYear &^ exempt
		nursingCharged := nursingMonths.Intersect(charged)

		for month := 1; month <= date.MonthsAYear; month++ {
			standard := in.HealthGrades.Lookup(monthlyPayOn(in.Pay, y, month, pay))
			row.HealthStandard = standard
			if !charged.Has(month) {
				continue
			}
			row.HealthOnPay += in.Rates.HealthPremium(standard, year)
			if nursingCharged.Has(month) {
				row.NursingCareOnPay += in.Rates.NursingCarePremium(standard, year)
			}
		}

		for month := 1; month <= date.MonthsAYear; month++ {
			graded := in.PensionGrades.Lookup(monthlyPayOn(in.Pay, y, month, pay))
			standard := min(graded, law.PensionStandardRemunerationCeilingOnPayslip(y, month))
			row.PensionStandard = standard
			if !charged.Has(month) {
				continue
			}
			row.PensionOnPay += law.PensionInsurancePremium(standard)
		}

		var healthPaidSoFar money.Yen
		for range pay.BonusesAYear {
			gross := pay.BonusPayment()

			healthStandard := law.HealthStandardBonus(healthPaidSoFar, gross)
			healthPaidSoFar += healthStandard
			row.HealthOnBonus += in.Rates.HealthPremium(healthStandard, year)
			if row.NursingCareInsured {
				row.NursingCareOnBonus += in.Rates.NursingCarePremium(healthStandard, year)
			}

			row.PensionOnBonus += law.PensionInsurancePremium(law.PensionStandardBonus(gross))
		}

		row.Health = row.HealthOnPay + row.HealthOnBonus
		row.Nursing = row.NursingCareOnPay + row.NursingCareOnBonus
		row.Pension = row.PensionOnPay + row.PensionOnBonus

		if row, err = readRecorded(row, in.Recorded, y); err != nil {
			return empty, fmt.Errorf("table.SocialInsuranceTable: %w", err)
		}

		rows = append(rows, relation.Row[SocialInsuranceRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}

func assertNoSplitCoverWithPay(y date.Year, bornOn date.Date, pay Pay) error {
	months := law.LateElderlyMonthsIn(y, bornOn)
	if months == date.NoMonths || months == date.WholeYear {
		return nil
	}
	if pay.Salary <= 0 {
		return nil
	}
	return fmt.Errorf(
		"%d 年は %v が 後期高齢者医療 で残りが被用者保険という割れた年なのに、"+
			"給与 %d 円がある。**この表は年に 1 つの制度しか持てず**、長いほうを"+
			"採るので短いほうの月がまるごと落ちる。"+
			"75 歳になる年に給与を置くなら、行が割れた年を持てるようにすること",
		y, months, pay.Salary)
}

func monthlyPayOn(payByYear relation.Table[Pay], y date.Year, month int, thisYear Pay) money.Yen {
	decided, ok := payByYear.At(law.RegularDecisionYearOnPayslip(y, month))
	if !ok {
		return thisYear.Monthly()
	}
	return decided.Monthly()
}

func readRecorded(row SocialInsuranceRow, recorded relation.Table[actuals.PremiumsDeducted], y date.Year) (SocialInsuranceRow, error) {
	deducted, ok := recorded.At(y)
	if !ok {
		return row, nil
	}

	if row.NursingCareInsured {
		return row, fmt.Errorf(
			"%d 年は介護保険の第2号の月がある年なのに、給与明細から保険料を読もうとしている。"+
				"**明細は健康保険料と介護保険料を 1 つの数で印字する**ので分けられず、"+
				"読むと介護のぶんが健保として入ったうえでモデルの介護保険料も足される。"+
				"この年を読む対象から外すか、明細に介護保険料の欄を足すこと", y)
	}

	if row.Cover != law.EmployeesHealthInsurance && (deducted.Health() != 0 || deducted.Pension() != 0) {
		return row, fmt.Errorf(
			"%d 年は被用者保険の年ではない（%v）のに、給与明細に健康保険料 %d 円・"+
				"厚生年金保険料 %d 円がある",
			y, row.Cover, deducted.Health(), deducted.Pension())
	}

	row.HealthOnPay, row.HealthOnBonus = deducted.HealthOnPay, deducted.HealthOnBonus
	row.PensionOnPay, row.PensionOnBonus = deducted.PensionOnPay, deducted.PensionOnBonus
	row.Health, row.Pension = deducted.Health(), deducted.Pension()
	row.EmploymentInsurance = deducted.Employment
	row.Recorded = true
	return row, nil
}
