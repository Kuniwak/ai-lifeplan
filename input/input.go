package input

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	PlanSlot tsv.Slot = "plan"

	HouseholdSlot tsv.Slot = "household"

	DisabilitySlot tsv.Slot = "disability"

	ResidenceSlot tsv.Slot = "residence"

	InvestmentReturnSlot tsv.Slot = "investment_return"

	FinancialCrisisSlot tsv.Slot = "financial_crisis"

	InflationSlot tsv.Slot = "inflation"

	RealWageGrowthSlot tsv.Slot = "real_wage_growth"

	CostGrowthSlot tsv.Slot = "cost_growth"

	PensionLevelSlot tsv.Slot = "pension_level"

	InflationTargetSlot tsv.Slot = "inflation_target"

	ReferenceRangeSlot tsv.Slot = "reference_range"

	PropertyAssessmentSlot tsv.Slot = "property_assessment"
)

const (
	IncomeHusbandSlot tsv.Slot = "income_husband"

	IncomeWifeSlot tsv.Slot = "income_wife"

	PensionSlot tsv.Slot = "pension"

	SchoolingSlot tsv.Slot = "schooling"

	TuitionSlot tsv.Slot = "tuition"

	ChildLivingCostSlot tsv.Slot = "child_living_cost"

	LivingCostSlot tsv.Slot = "living_cost"

	AllowanceHusbandSlot tsv.Slot = "allowance_husband"
	AllowanceWifeSlot    tsv.Slot = "allowance_wife"

	MedicalExpenseSlot tsv.Slot = "medical_expense"

	MedicalCostByAgeSlot tsv.Slot = "medical_cost_by_age"

	ExtraordinaryCostSlot tsv.Slot = "extraordinary_cost"

	LifeInsurancePremiumSlot tsv.Slot = "life_insurance_premium"

	MutualAidContributionSlot tsv.Slot = "mutual_aid_contribution"

	PropertyInsuranceSlot tsv.Slot = "property_insurance"

	HousingSlot tsv.Slot = "housing"

	LoanSlot tsv.Slot = "loan"

	LoanSettlementSlot tsv.Slot = "loan_settlement"

	LastResortSlot tsv.Slot = "last_resort"

	HousingRentSlot tsv.Slot = "housing_rent"

	HousingMaintenanceSlot tsv.Slot = "housing_maintenance"

	InvestmentSlot tsv.Slot = "investment"

	BalanceSlot tsv.Slot = "balance"

	CashflowSlot tsv.Slot = "cashflow"

	AdjustmentsSlot tsv.Slot = "adjustments"
)

var requiredSlots = []tsv.Slot{
	PlanSlot,
	HouseholdSlot,
	DisabilitySlot,
	ResidenceSlot,
	InvestmentReturnSlot,
	FinancialCrisisSlot,
	InflationSlot,
	RealWageGrowthSlot,
	CostGrowthSlot,
	PensionLevelSlot,
	InflationTargetSlot,
	ReferenceRangeSlot,
	PropertyAssessmentSlot,

	IncomeHusbandSlot,
	IncomeWifeSlot,
	PensionSlot,
	SchoolingSlot,
	TuitionSlot,
	ChildLivingCostSlot,
	LivingCostSlot,
	AllowanceHusbandSlot,
	AllowanceWifeSlot,
	MedicalExpenseSlot,
	MedicalCostByAgeSlot,
	ExtraordinaryCostSlot,
	LifeInsurancePremiumSlot,
	MutualAidContributionSlot,
	PropertyInsuranceSlot,
	HousingSlot,
	LoanSlot,
	LoanSettlementSlot,
	LastResortSlot,
	HousingRentSlot,
	HousingMaintenanceSlot,
	InvestmentSlot,

	BalanceSlot,
	CashflowSlot,
	AdjustmentsSlot,
}

func RequiredSlots() []tsv.Slot {
	names := slices.Clone(requiredSlots)
	slices.Sort(names)
	return names
}

func Load(root string, paths map[tsv.Slot]string) (map[tsv.Slot]*tsv.Table, error) {
	slots := make([]tsv.Slot, 0, len(paths))
	for slot := range paths {
		slots = append(slots, slot)
	}
	slices.Sort(slots)

	tables := make(map[tsv.Slot]*tsv.Table, len(slots))
	var errs []error
	for _, slot := range slots {
		path := tsv.Under(root, paths[slot])

		table, err := tsv.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("input.Load: %s: %w", slot, err))
			continue
		}
		tables[slot] = table
	}

	return tables, errors.Join(errs...)
}
