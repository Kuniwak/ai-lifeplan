package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type HouseholdInsuranceRow struct {
	CoverOf []PersonCover

	Kokuho   money.Yen
	KokuhoOf []PersonKokuho

	Kouki   money.Yen
	KoukiOf []PersonPremium

	NationalPension   money.Yen
	NationalPensionOf []PersonPremium

	NursingCare   money.Yen
	NursingCareOf []NursingCarePremium
}

type PersonCover struct {
	Name PersonName

	Under []law.CoverMonths
}

func (c PersonCover) Months(cover law.Cover) date.Months {
	for _, under := range c.Under {
		if under.Cover == cover {
			return under.Months
		}
	}
	return date.NoMonths
}

func (c PersonCover) Longest() law.Cover {
	return law.LongestCover(c.Under)
}

type PersonKokuho struct {
	Name PersonName
	law.KokuhoMember
}

func (r HouseholdInsuranceRow) KokuhoOfPerson(name PersonName) (PersonKokuho, bool) {
	for _, m := range r.KokuhoOf {
		if m.Name == name {
			return m, true
		}
	}
	return PersonKokuho{}, false
}

type PersonPremium struct {
	Name    PersonName
	Premium money.Yen

	SpeciallyCollected bool
}

type NursingCarePremium struct {
	PersonPremium
	Stage law.NursingCareStage
}

func (r HouseholdInsuranceRow) DeductionsBy(earner PersonName) map[PersonName]money.Yen {
	by := map[PersonName]money.Yen{earner: r.Kokuho}
	for _, split := range [][]PersonPremium{r.KoukiOf, r.NursingCarePremiums(), r.NationalPensionOf} {
		for _, premium := range split {
			claimant := earner
			if premium.SpeciallyCollected {
				claimant = premium.Name
			}
			by[claimant] += premium.Premium
		}
	}
	return by
}

func (r HouseholdInsuranceRow) NursingCarePremiums() []PersonPremium {
	premiums := make([]PersonPremium, 0, len(r.NursingCareOf))
	for _, premium := range r.NursingCareOf {
		premiums = append(premiums, premium.PersonPremium)
	}
	return premiums
}

func (r HouseholdInsuranceRow) NursingCareOfPerson(name PersonName) (NursingCarePremium, bool) {
	for _, premium := range r.NursingCareOf {
		if premium.Name == name {
			return premium, true
		}
	}
	return NursingCarePremium{}, false
}

func (r HouseholdInsuranceRow) CoverOfPerson(name PersonName) (PersonCover, bool) {
	for _, c := range r.CoverOf {
		if c.Name == name {
			return c, true
		}
	}
	return PersonCover{}, false
}

func (r HouseholdInsuranceRow) Total() money.Yen {
	return r.Kokuho + r.Kouki + r.NationalPension
}

type HouseholdInsuranceInput struct {
	Calendar relation.Table[CalendarRow]

	Members map[PersonName]relation.Table[HouseholdMemberYear]

	EmployeeCovers map[PersonName]relation.Table[law.Cover]

	Disabilities Disabilities

	Kokuho          law.KokuhoTable
	Kouki           law.KoukiRatesTable
	NationalPension law.NationalPensionPremiumTable
	NursingCare     law.NursingCarePremiumTable
}

func HouseholdInsuranceTable(in HouseholdInsuranceInput) (relation.Table[HouseholdInsuranceRow], error) {
	var empty relation.Table[HouseholdInsuranceRow]

	years := in.Calendar.Years()
	rows := make([]relation.Row[HouseholdInsuranceRow], 0, len(years))

	lastRegular := make(map[PersonName]law.LastRegularInstalments, len(years))

	for _, y := range years {
		calendar, _ := in.Calendar.At(y)

		if len(in.EmployeeCovers) == 0 {
			return empty, fmt.Errorf(
				"table.HouseholdInsuranceTable: EmployeeCovers が空である。" +
					"勤めていない人も自分の表を持つ（table.SocialInsuranceTable）")
		}

		covers := make(map[PersonName]law.Cover, len(in.EmployeeCovers))
		for name, byYear := range in.EmployeeCovers {
			cover, ok := byYear.At(y)
			if !ok {
				return empty, fmt.Errorf("table.HouseholdInsuranceTable: nothing is known about %q's cover in %d", name, y)
			}
			covers[name] = cover
		}

		members := make(map[PersonName]HouseholdMemberYear, len(calendar.Ages))
		for _, person := range calendar.Ages {
			member, ok := in.Members[person.Name].At(y)
			if !ok {
				return empty, fmt.Errorf(
					"table.HouseholdInsuranceTable: %d 年の %q の収入が渡されていない。"+
						"子どもなら 0 円の行が要る（table.HouseholdMembersOf）", y, person.Name)
			}
			members[person.Name] = member
		}

		for name := range covers {
			if _, ok := members[name]; !ok {
				return empty, fmt.Errorf(
					"table.HouseholdInsuranceTable: %d 年の暦に被保険者 %q がいない", y, name)
			}
		}

		receipts := make(map[PersonName]money.Yen, len(covers))
		for name := range covers {
			receipts[name] = members[name].Receipts
		}
		principal, hasPrincipal := law.PrincipalInsured(covers, receipts)
		insured := members[principal]

		var row HouseholdInsuranceRow

		householdTaxed := false
		for _, person := range calendar.Ages {
			if !calendar.InHousehold(person.Name) {
				continue
			}
			if members[person.Name].Taxed.PerCapita {
				householdTaxed = true
			}
		}

		for _, person := range calendar.Ages {
			member := members[person.Name]
			isEmployee := covers[person.Name] == law.EmployeesHealthInsurance
			cover := PersonCover{Name: person.Name}
			var kouki money.Yen

			var otherwise law.Cover
			switch {
			case !calendar.InHousehold(person.Name):
				otherwise = law.NoCover

			case isEmployee:
				otherwise = covers[person.Name]

			case law.HealthDependant(covers[principal], insured.Receipts, member.Receipts, person.Age,
				in.Disabilities.DisabilityPensionFor(person.Name)):
				otherwise = law.EmployeesHealthInsurance

			default:
				otherwise = law.NationalHealthInsurance
			}
			cover.Under = law.CoverMonthsIn(y, person.BornOn, otherwise)

			for _, under := range cover.Under {
				switch under.Cover {
				case law.NationalHealthInsurance:
					row.KokuhoOf = append(row.KokuhoOf, PersonKokuho{
						Name: person.Name,
						KokuhoMember: law.KokuhoMember{
							Base:   max(member.Income-law.KokuhoBasicDeduction, 0),
							Months: under.Months,
							NursingCareMonths: under.Months.Intersect(
								law.NursingCareSecondCategoryMonthsIn(y, person.BornOn)),
						},
					})

				case law.LateElderlyHealthCare:
					kouki = in.Kouki.Premium(member.Income, y).ForMonths(under.Months.Count())
					row.Kouki += kouki
				}
			}

			row.CoverOf = append(row.CoverOf, cover)

			nursingMonths := law.NursingCareFirstCategoryMonthsIn(y, person.BornOn)

			var nursingCare money.Yen
			var stage law.NursingCareStage
			if calendar.InHousehold(person.Name) && nursingMonths != date.NoMonths {
				subject := law.NursingCarePremiumSubject{
					Taxed:           member.Taxed.PerCapita,
					HouseholdTaxed:  householdTaxed,
					TotalIncome:     member.Income,
					PensionReceipts: member.PensionReceipts,
					PensionIncome:   member.PensionIncome,
				}
				charge := in.NursingCare.Charge(subject, y)
				nursingCare, stage = charge.Premium.ForMonths(nursingMonths.Count()), charge.Stage
				row.NursingCare += nursingCare
			}

			subject := law.SpecialCollectionSubject{
				Pension:        member.OldAgePensionBenefit,
				NursingCare:    nursingCare,
				LateElderly:    kouki,
				JoinedThisYear: person.Age == law.NursingCareFirstCategoryAge,
			}.From(lastRegular[person.Name])

			withheld := law.SpeciallyCollectedFrom(subject)
			lastRegular[person.Name] = subject.Next(withheld)
			if kouki > 0 {
				row.KoukiOf = append(row.KoukiOf, PersonPremium{
					Name: person.Name, Premium: kouki, SpeciallyCollected: withheld.LateElderly})
			}
			if nursingCare > 0 {
				row.NursingCareOf = append(row.NursingCareOf, NursingCarePremium{
					PersonPremium: PersonPremium{
						Name: person.Name, Premium: nursingCare,
						SpeciallyCollected: withheld.NursingCare,
					},
					Stage: stage,
				})
			}

			if months := law.NationalPensionMonthsIn(y, person.BornOn, cover.Longest(),
				isEmployee, person.Relation == Spouse && hasPrincipal).Count(); months > 0 {
				premium := in.NationalPension.Monthly(y) * money.Yen(months)
				row.NationalPension += premium
				row.NationalPensionOf = append(row.NationalPensionOf, PersonPremium{Name: person.Name, Premium: premium})
			}
		}

		household := law.KokuhoHousehold{Members: make([]law.KokuhoMember, 0, len(row.KokuhoOf))}
		for _, share := range row.KokuhoOf {
			household.Members = append(household.Members, share.KokuhoMember)
		}

		premium, err := in.Kokuho.Premium(household, y)
		if err != nil {
			return empty, fmt.Errorf("table.HouseholdInsuranceTable: %d: %w", y, err)
		}
		row.Kokuho = premium

		rows = append(rows, relation.Row[HouseholdInsuranceRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}
