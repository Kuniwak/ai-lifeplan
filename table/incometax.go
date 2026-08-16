package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type Deductions struct {
	Basic           law.Deduction
	Spouse          law.Deduction
	Dependents      law.Deduction
	Disability      law.Deduction
	SocialInsurance money.Yen

	MutualAid money.Yen

	LifeInsurance law.Deduction
	Earthquake    law.Deduction
	Medical       money.Yen

	DependentCount int

	RelativeCount int

	SpouseGap money.Yen

	SpouseSameLivelihood bool
}

func (d Deductions) Total() law.Deduction {
	return law.Deduction{
		IncomeTax: d.Basic.IncomeTax + d.Spouse.IncomeTax + d.Dependents.IncomeTax +
			d.Disability.IncomeTax + d.SocialInsurance + d.MutualAid +
			d.LifeInsurance.IncomeTax + d.Earthquake.IncomeTax + d.Medical,
		Resident: d.Basic.Resident + d.Spouse.Resident + d.Dependents.Resident +
			d.Disability.Resident + d.SocialInsurance + d.MutualAid +
			d.LifeInsurance.Resident + d.Earthquake.Resident + d.Medical,
	}
}

func (d Deductions) HumanDeductionGap() money.Yen {
	gap := func(x law.Deduction) money.Yen { return max(x.IncomeTax-x.Resident, 0) }
	return law.BasicHumanDeductionGap + d.SpouseGap + gap(d.Dependents) + gap(d.Disability)
}

type IncomeTaxRow struct {
	TotalIncome money.Yen

	Deductions Deductions

	Disabled bool

	Taxable money.Yen

	Tax money.Yen

	HousingLoanBalance, HousingLoanCredit money.Yen

	SpecialCredit money.Yen

	BaseTax, Surtax, Payable money.Yen
}

type IncomeTaxInput struct {
	Calendar relation.Table[CalendarRow]

	Taxpayer, Spouse PersonName

	Income map[PersonName]relation.Table[money.Yen]

	SocialInsurance relation.Table[money.Yen]

	LifeInsurancePremium, MedicalInsurancePremium, EarthquakePremium relation.Table[money.Yen]

	MutualAidContribution relation.Table[money.Yen]

	MedicalPaid, MedicalRefunded relation.Table[money.Yen]

	Disabilities Disabilities

	ClaimsDependents bool

	HousingLoanBalance relation.Table[money.Yen]
	MovedIn            *date.Year

	DisabilityDeduction law.DisabilityDeductionTable
	HousingLoanCredit   law.HousingLoanCreditTable

	SpouseIncomeCeiling law.SpouseIncomeCeilingTable
}

type Disability struct {
	Category    law.DisabilityCategoryValue
	CertifiedIn date.Year

	DisabilityPension law.DisabilityPensionEligible
}

func (d Disability) AppliesIn(y date.Year) bool { return y >= d.CertifiedIn }

type Disabilities map[PersonName]Disability

func (d Disabilities) AppliesTo(name PersonName, y date.Year) bool {
	certified, ok := d[name]
	return ok && certified.AppliesIn(y)
}

func (d Disabilities) DisabilityPensionFor(name PersonName) law.DisabilityPensionEligible {
	certified, ok := d[name]
	if !ok {
		return law.DisabilityPensionNo
	}
	return certified.DisabilityPension
}

func IncomeTaxTable(in IncomeTaxInput) (relation.Table[IncomeTaxRow], error) {
	var empty relation.Table[IncomeTaxRow]

	years := in.Calendar.Years()
	rows := make([]relation.Row[IncomeTaxRow], 0, len(years))

	if in.MovedIn != nil {
		if _, ok := in.HousingLoanCredit.Terms(*in.MovedIn); !ok {
			return empty, fmt.Errorf(
				"table.IncomeTaxTable: %d 年の居住開始に当たる住宅借入金等特別控除の条件が data/law に無い。表の外の年を 0 円として通すと、控除期間を使い切った世帯と区別がつかなくなる",
				*in.MovedIn)
		}
	}

	for _, y := range years {
		calendar, _ := in.Calendar.At(y)

		income, ok := in.Income[in.Taxpayer].At(y)
		if !ok {
			return empty, fmt.Errorf("table.IncomeTaxTable: no income for %q in %d", in.Taxpayer, y)
		}

		row := IncomeTaxRow{TotalIncome: income}
		row.Disabled = in.Disabilities.AppliesTo(in.Taxpayer, y)
		d, err := deductionsFor(in, calendar, y, income)
		if err != nil {
			return empty, err
		}
		row.Deductions = d

		if balance, ok := in.HousingLoanBalance.At(y); ok {
			row.HousingLoanBalance = balance
		}
		if in.MovedIn != nil {
			row.HousingLoanCredit = in.HousingLoanCredit.Credit(
				row.HousingLoanBalance, income, *in.MovedIn, y)
		}

		chained := ChainTax(income, d.Total().IncomeTax, row.HousingLoanCredit, d.DependentCount, y)
		row.Taxable, row.Tax = chained.Taxable, chained.Tax
		row.SpecialCredit = chained.SpecialCredit
		row.BaseTax, row.Surtax, row.Payable = chained.BaseTax, chained.Surtax, chained.Payable

		rows = append(rows, relation.Row[IncomeTaxRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}

func deductionsFor(in IncomeTaxInput, calendar CalendarRow, y date.Year, income money.Yen) (Deductions, error) {
	var d Deductions

	d.Basic = law.Deduction{
		IncomeTax: law.BasicDeduction(income, y),
		Resident:  law.ResidentBasicDeduction(income, y),
	}

	if in.Spouse != "" {
		spouseIncome, _ := in.Income[in.Spouse].At(y)
		spouseAge, _ := calendar.AgeOf(in.Spouse)

		ceiling := in.SpouseIncomeCeiling.Ceiling(y)
		d.Spouse, d.SpouseGap = law.SpouseDeductionsOf(spouseIncome, income, spouseAge, ceiling, y)
	}

	dependents := DependentsOf(dependentsInputOf(in), calendar, y)
	d.DependentCount = dependents.Count()
	d.RelativeCount = len(dependents.Relatives)
	d.SpouseSameLivelihood = dependents.SpouseSameLivelihood

	for _, person := range dependents.Relatives {
		dependent := law.DependentDeduction(person.Age, true)
		d.Dependents.IncomeTax += dependent.IncomeTax
		d.Dependents.Resident += dependent.Resident
	}

	for name, disability := range in.Disabilities {
		if !disability.AppliesIn(y) {
			continue
		}
		switch {
		case name == in.Taxpayer:
		case !in.ClaimsDependents:
			continue
		case name == in.Spouse && d.SpouseSameLivelihood:
		case dependents.Includes(name):
		default:
			continue
		}

		amount, err := in.DisabilityDeduction.Lookup(disability.Category)
		if err != nil {
			return d, fmt.Errorf("table.IncomeTaxTable: %d: %s: %w", y, name, err)
		}
		d.Disability.IncomeTax += amount.IncomeTax
		d.Disability.Resident += amount.Resident
	}

	if paid, ok := in.SocialInsurance.At(y); ok {
		d.SocialInsurance = paid
	}
	if paid, ok := in.MutualAidContribution.At(y); ok {
		d.MutualAid = paid
	}
	{
		general, _ := in.LifeInsurancePremium.At(y)
		medical, _ := in.MedicalInsurancePremium.At(y)
		const noAnnuity money.Yen = 0
		d.LifeInsurance = law.LifeInsuranceDeductionTotal(
			general, medical, noAnnuity, y, hasYoungDependant(calendar))
	}
	if premium, ok := in.EarthquakePremium.At(y); ok {
		d.Earthquake = law.EarthquakeInsuranceDeduction(premium)
	}

	paid, _ := in.MedicalPaid.At(y)
	refunded, _ := in.MedicalRefunded.At(y)
	d.Medical = law.MedicalDeduction(paid, refunded, income)

	return d, nil
}

func dependentsInputOf(in IncomeTaxInput) DependentsInput {
	return DependentsInput{
		Taxpayer:            in.Taxpayer,
		Spouse:              in.Spouse,
		Income:              in.Income,
		SpouseIncomeCeiling: in.SpouseIncomeCeiling,
		ClaimsDependents:    in.ClaimsDependents,
	}
}

type ChainedTax struct {
	Taxable money.Yen

	Tax money.Yen

	AfterCredits money.Yen

	SpecialCredit money.Yen

	BaseTax money.Yen

	Surtax money.Yen

	Payable money.Yen
}

func ChainTax(totalIncome, deductions, credits money.Yen, sameLivelihoodDependants int, year date.Year) ChainedTax {
	var out ChainedTax

	out.Taxable = max(totalIncome-deductions, 0).TruncateTaxableIncome()
	out.Tax = law.IncomeTax(out.Taxable)

	out.AfterCredits = max(out.Tax-credits, 0)

	out.SpecialCredit = min(law.SpecialTaxCredit(year, totalIncome, sameLivelihoodDependants), out.AfterCredits)

	out.BaseTax = out.AfterCredits - out.SpecialCredit
	out.Surtax = law.ReconstructionSurtax(out.BaseTax, year)
	out.Payable = out.BaseTax + out.Surtax
	return out
}
