package input

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

type Kind int

const (
	Step Kind = iota

	Events

	Lookup
)

const YearColumn tsv.ColumnName = "西暦"

type Shape struct {
	Slot tsv.Slot

	Kind Kind

	YearColumn tsv.ColumnName

	OrderedBy tsv.ColumnName

	Columns []validate.Column

	KeyColumns []tsv.ColumnName
}

func yen(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitYen, Parse: validate.AsYen}
}
func count(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitCount, Parse: validate.AsCount}
}
func year(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitYear, Parse: validate.AsYear}
}
func percent(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitPercent, Parse: validate.AsPercent}
}

func priceMove(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitPriceMove, Parse: validate.AsPriceMove}
}

func percentAtLeast(name tsv.ColumnName, num, den int64) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitPercent, Parse: validate.AsPercentAtLeast(num, den)}
}

func yenAtMost(name tsv.ColumnName, ceiling money.Yen) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitYen, Parse: validate.AsYenAtMost(ceiling)}
}

func yenOnly(name tsv.ColumnName, only money.Yen, because string) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitYen, Parse: validate.AsYenOnly(only, because)}
}

func monthsColumn(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitMonths, Parse: validate.AsMonths}
}
func dateColumn(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitDate, Parse: validate.AsDate}
}
func text(name tsv.ColumnName) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitText, Parse: validate.AsText}
}

func oneOf(name tsv.ColumnName, words ...string) validate.Column {
	return validate.Column{Name: name, Unit: validate.UnitText, Parse: validate.AsOneOf(words...)}
}

const (
	PlanStartColumn tsv.ColumnName = "開始年"
	PlanEndColumn   tsv.ColumnName = "終了年"
)

const (
	InflationRateColumn tsv.ColumnName = "インフレ率"

	RealWageGrowthColumn tsv.ColumnName = "実質賃金上昇率"

	MedicalCostGrowthColumn tsv.ColumnName = "医療費上昇率"

	NursingCarePremiumGrowthColumn tsv.ColumnName = "介護保険料上昇率"

	AgeFromColumn    tsv.ColumnName = "下限年齢"
	MedicalCostAtAge tsv.ColumnName = "1人当たり医療費[円/年]"
	OutOfPocketAtAge tsv.ColumnName = "1人当たり自己負担額[円/年]"

	NursingCareCostGrowthColumn tsv.ColumnName = "介護費上昇率"

	PensionBasicLevelColumn        tsv.ColumnName = "基礎年金の水準"
	PensionProportionalLevelColumn tsv.ColumnName = "報酬比例年金の水準"

	PricedItemColumn tsv.ColumnName = "項目"

	InflationRatioColumn tsv.ColumnName = "一般物価に対する比または差"
)

var InflationRatioFloor = money.NewRate(0, 1)

const DialColumn tsv.ColumnName = "ダイヤル"

const LoanSettledInColumn tsv.ColumnName = "一括返済年"

type Settlement string

const (
	LoanNoSettlement Settlement = "しない"
)

func AsYearOrNever(field string) error {
	if Settlement(field) == LoanNoSettlement {
		return nil
	}
	if err := validate.AsYear(field); err != nil {
		return fmt.Errorf("%w（%q と書けば一括返済しない）", err, LoanNoSettlement)
	}
	return nil
}

type PricedItem string

const (
	CoupleLivingItem   PricedItem = "夫婦生活費"
	ChildLivingItem    PricedItem = "子生活費"
	MedicalItem        PricedItem = "医療費"
	AllowanceItem      PricedItem = "小遣い"
	ExtraordinaryItem  PricedItem = "臨時費用"
	EducationItem      PricedItem = "教育費"
	LifeInsuranceItem  PricedItem = "生命保険料"
	FireInsuranceItem  PricedItem = "火災保険料"
	QuakeInsuranceItem PricedItem = "地震保険料"
	RentItem           PricedItem = "家賃"
	DepositItem        PricedItem = "頭金"
	LoanPaidItem       PricedItem = "ローン返済"
	MaintenanceItem    PricedItem = "住宅維持費"

	ContributionItem PricedItem = "積立額"
	CashFloorItem    PricedItem = "貯蓄維持目標"
	MutualAidItem    PricedItem = "小規模企業共済等掛金"
)

var PricedItems = []PricedItem{
	CoupleLivingItem, ChildLivingItem, MedicalItem, AllowanceItem, ExtraordinaryItem,
	EducationItem,
	LifeInsuranceItem, FireInsuranceItem, QuakeInsuranceItem,
	RentItem, DepositItem, LoanPaidItem, MaintenanceItem,
	ContributionItem, CashFloorItem, MutualAidItem,
}

func PricedItemNames() []string {
	names := make([]string, 0, len(PricedItems))
	for _, item := range PricedItems {
		names = append(names, string(item))
	}
	return names
}

var shapes = []Shape{
	{
		Slot: PlanSlot, Kind: Lookup,
		Columns: []validate.Column{year(PlanStartColumn), year(PlanEndColumn)},
	},
	{
		Slot: HouseholdSlot, Kind: Lookup,
		Columns: []validate.Column{
			text(PersonColumn),
			oneOf(RelationColumn, RelationWords...),
			dateColumn(BornOnColumn),
		},
	},
	{
		Slot: DisabilitySlot, Kind: Lookup,
		Columns: []validate.Column{
			text(DisabledPersonColumn), year(DisabilityCertified), text(DisabilityCategory),
			oneOf(DisabilityPensionColumn, DisabilityPensionWords...),
		},
	},
	{
		Slot: ResidenceSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), text(MunicipalityColumn)},
	},
	{
		Slot: InvestmentReturnSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), percent(ReturnColumn)},
	},
	{
		Slot: FinancialCrisisSlot, Kind: Events, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), percent(CrashColumn)},
	},

	{
		Slot: InflationSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), percent(InflationRateColumn)},
	},
	{
		Slot: CostGrowthSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{
			year(YearColumn), percent(MedicalCostGrowthColumn),
			percent(NursingCareCostGrowthColumn), percent(NursingCarePremiumGrowthColumn),
		},
	},
	{
		Slot: RealWageGrowthSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), percent(RealWageGrowthColumn)},
	},
	{
		Slot: PensionLevelSlot, Kind: Lookup,
		Columns: []validate.Column{
			year(YearColumn), percent(PensionBasicLevelColumn), percent(PensionProportionalLevelColumn),
		},
	},
	{
		Slot: InflationTargetSlot, Kind: Lookup,
		KeyColumns: []tsv.ColumnName{PricedItemColumn},
		Columns: []validate.Column{
			text(PricedItemColumn),
			priceMove(InflationRatioColumn),
			text("理由"),
		},
	},

	{
		Slot: ReferenceRangeSlot, Kind: Lookup,
		Columns: []validate.Column{
			text(DialColumn), percent("下限"), percent("上限"), text(SourceColumn),
		},
	},

	{
		Slot: PropertyAssessmentSlot, Kind: Lookup,
		Columns: []validate.Column{year(AssessedYearColumn), yen(LandValueColumn), yen(HouseBaseColumn), text(SourceColumn)},
	},

	{
		Slot: IncomeHusbandSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{
			year(YearColumn),
			yen(AnnualSalaryColumn), yen(BonusColumn), count(BonusesAYearColumn),
			count(WeeklyHoursColumn),
			count(NormalWeeklyHoursColumn),
			oneOf(SpecifiedWorkplaceColumn, SpecifiedWorkplaceWords...),
			count(LeaveMonthsColumn), monthsColumn(ExemptMonthsColumn),
			yen(BusinessReceiptsColumn), yen(BusinessExpensesColumn),
			yenAtMost(SmallDepreciableColumn, SmallDepreciableYearlyLimit),
			oneOf(BlueFormRecordKeepingColumn, BlueFormRecordKeepingWords...),
			yen(MiscellaneousReceiptsColumn),
		},
	},
	{
		Slot: IncomeWifeSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{
			year(YearColumn),
			yen(AnnualSalaryColumn), yen(BonusColumn), count(BonusesAYearColumn),
			count(WeeklyHoursColumn),
			count(NormalWeeklyHoursColumn),
			oneOf(SpecifiedWorkplaceColumn, SpecifiedWorkplaceWords...),
			yen(BusinessReceiptsColumn), yen(BusinessExpensesColumn),
			yenAtMost(SmallDepreciableColumn, SmallDepreciableYearlyLimit),
			oneOf(BlueFormRecordKeepingColumn, BlueFormRecordKeepingWords...),
			count(LeaveMonthsColumn),
		},
	},
	{
		Slot: PensionSlot, Kind: Lookup,
		Columns: []validate.Column{
			text(PersonColumn), year(PensionStartColumn),
			percent(PensionExpectedColumn), text(SourceColumn),
		},
	},
	{
		Slot: SchoolingSlot, Kind: Lookup, OrderedBy: StageFromAgeColumn,
		Columns: []validate.Column{text(StageColumn), count(StageFromAgeColumn)},
	},
	{
		Slot: TuitionSlot, Kind: Lookup,
		Columns: []validate.Column{text(StageColumn), yen(TuitionColumn)},
	},
	{
		Slot: ChildLivingCostSlot, Kind: Lookup,
		Columns: []validate.Column{text(StageColumn), yen(ChildLivingColumn)},
	},
	{
		Slot: LivingCostSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(CoupleLivingColumn)},
	},
	{
		Slot: AllowanceHusbandSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(AllowanceColumn)},
	},
	{
		Slot: AllowanceWifeSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(AllowanceColumn)},
	},
	{
		Slot: MedicalCostByAgeSlot, Kind: Lookup,
		Columns: []validate.Column{
			count(AgeFromColumn), yen(MedicalCostAtAge), yen(OutOfPocketAtAge), text(SourceColumn),
		},
	},
	{
		Slot: MedicalExpenseSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(MedicalColumn), yen(MedicalRefundColumn), text(SourceColumn)},
	},
	{
		Slot: ExtraordinaryCostSlot, Kind: Events, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), text("名目"), yen(ExtraordinaryColumn)},
	},
	{
		Slot: LifeInsurancePremiumSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(LifeInsuranceColumn), yen(MedicalInsuranceColumn)},
	},
	{
		Slot: MutualAidContributionSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(MutualAidContributionColumn)},
	},
	{
		Slot: PropertyInsuranceSlot, Kind: Events, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(FireInsuranceColumn), yen(QuakeInsuranceColumn), count(InsuranceTermColumn)},
	},
	{
		Slot: HousingSlot, Kind: Events, YearColumn: "取得年",
		Columns: []validate.Column{year(BoughtInColumn), yen(DepositColumn), text(SourceColumn)},
	},
	{
		Slot: LoanSlot, Kind: Lookup,
		KeyColumns: []tsv.ColumnName{LoanNameColumn},
		Columns: []validate.Column{
			text(LoanNameColumn), year(LoanDrawnInColumn), year(LoanFirstYearColumn), count(LoanFirstMonthColumn),
			yen(PrincipalColumn), count(LoanYearsColumn), percent(LoanRateColumn),
			count(LoanFixedYearsColumn), percent(LoanFloatingColumn), text(SourceColumn),
		},
	},
	{
		Slot: LoanSettlementSlot, Kind: Lookup,
		Columns: []validate.Column{
			{Name: LoanSettledInColumn, Unit: validate.UnitText, Parse: AsYearOrNever},
		},
	},
	{
		Slot: LastResortSlot, Kind: Lookup,
		Columns: []validate.Column{
			text(MeasureColumn), count(MeasureFromAge), percent(MeasureProceedRate),
			percent(MeasureInterest), yen(MeasureRentMonthly), text(MeasureGivesUpHome),
			text(SourceColumn),
		},
	},
	{
		Slot: HousingRentSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(RentColumn)},
	},
	{
		Slot: HousingMaintenanceSlot, Kind: Events, YearColumn: YearColumn,
		Columns: []validate.Column{year(YearColumn), yen(MaintenanceColumn)},
	},
	{
		Slot: InvestmentSlot, Kind: Step, YearColumn: YearColumn,
		Columns: []validate.Column{
			year(YearColumn), yen("積立額[円/月]"), yen("貯蓄維持目標[円]"),
			yen("NISA生涯投資枠[円]"), text(SellNISAFirstColumn), text(SourceColumn),
		},
	},

	{
		Slot: BalanceSlot, Kind: Events,
		YearColumn: actuals.BalanceYearColumn,
		OrderedBy:  actuals.BalanceYearColumn,
		Columns: []validate.Column{
			year(actuals.BalanceYearColumn),
			yen(actuals.BalanceCashColumn),
			yen(actuals.BalanceInvestedColumn),
			yen(actuals.BalanceNISAColumn),
			yen(actuals.BalanceMaturingNISAColumn),
			yen(actuals.BalanceTaxableColumn),
			yen(actuals.BalanceBasisColumn),
			yen(actuals.BalanceLockedColumn),
			yen(actuals.BalanceTotalColumn),
			{
				Name:  actuals.BalancePartialColumn,
				Unit:  validate.UnitText,
				Parse: validate.AsOptional(validate.AsText),
			},
		},
	},

	{
		Slot: CashflowSlot, Kind: Lookup,
		Columns: []validate.Column{
			text(actuals.CashflowMonthColumn),
			text(actuals.CashflowItemColumn),
			yen(actuals.CashflowAmountColumn),
		},
	},

	{
		Slot: AdjustmentsSlot, Kind: Lookup,
		Columns: []validate.Column{
			text(actuals.AdjustmentMonthColumn),
			text(actuals.AdjustmentItemColumn),
			yen(actuals.AdjustmentAmountColumn),
			text(actuals.AdjustmentSourceColumn),
		},
	},
}

func Shapes() []Shape {
	described := slices.Clone(shapes)
	slices.SortFunc(described, func(a, b Shape) int {
		switch {
		case a.Slot < b.Slot:
			return -1
		case a.Slot > b.Slot:
			return 1
		default:
			return 0
		}
	})
	return described
}

func PlanSpan(table *tsv.Table) (from, to date.Year, err error) {
	if table == nil {
		return 0, 0, fmt.Errorf("input.PlanSpan: no plan table")
	}
	if len(table.Rows) != 1 {
		return 0, 0, fmt.Errorf("input.PlanSpan: want exactly one row, the span of the plan, got %d", len(table.Rows))
	}

	read := func(column tsv.ColumnName) (date.Year, error) {
		i, ok := table.ColumnIndex(column)
		if !ok {
			return 0, fmt.Errorf("input.PlanSpan: no %q column", column)
		}
		y, err := date.ParseYear(table.Rows[0][i])
		if err != nil {
			return 0, fmt.Errorf("input.PlanSpan: %q: %w", column, err)
		}
		return y, nil
	}

	if from, err = read(PlanStartColumn); err != nil {
		return 0, 0, err
	}
	if to, err = read(PlanEndColumn); err != nil {
		return 0, 0, err
	}
	if from > to {
		return 0, 0, fmt.Errorf("input.PlanSpan: the span runs backwards: from %d to %d", from, to)
	}
	return from, to, nil
}

func Rules(planStart date.Year) []validate.Rule {
	rules := []validate.Rule{
		validate.Scoped(validate.EveryChoiceMade(
			InflationTargetSlot, PricedItemColumn, InflationRatioColumn,
			PricedItemNames(), validate.AnyAnswer,
			"（率ですらない値）",
		), InflationTargetSlot),
		validate.Scoped(validate.LoanSettlementInTerm(
			LoanSlot, "初回返済年", "初回返済月", "借入期間[年]",
			LoanSettlementSlot, LoanSettledInColumn, string(LoanNoSettlement),
		), LoanSettlementSlot),
		validate.Scoped(validate.LoanFixedPeriod(
			LoanSlot, "借入期間[年]", "固定期間[年]",
		), LoanSlot),
	}

	bySlot := make(map[tsv.Slot][]validate.PositivePair)
	var slots []tsv.Slot
	for _, shape := range Shapes() {
		if len(shape.KeyColumns) > 0 {
			rules = append(rules, validate.Scoped(
				validate.UniqueKey(shape.Slot, shape.KeyColumns), shape.Slot))
		}
	}

	for _, pair := range PositiveTogether() {
		if _, seen := bySlot[pair.Slot]; !seen {
			slots = append(slots, pair.Slot)
		}
		bySlot[pair.Slot] = append(bySlot[pair.Slot], validate.PositivePair{
			Positive: pair.Positive, Needed: pair.Needed, Why: pair.Why,
		})
	}
	for _, slot := range slots {
		rules = append(rules, validate.Scoped(
			validate.PositiveNeeds(slot, YearColumn, bySlot[slot]), slot))
	}

	for _, shape := range Shapes() {
		rules = append(rules, validate.Scoped(validate.ColumnSchema(shape.Slot, shape.Columns), shape.Slot))

		if shape.OrderedBy != "" {
			rules = append(rules,
				validate.Scoped(validate.Ascending(shape.Slot, shape.OrderedBy), shape.Slot))
		}

		if shape.Kind != Step {
			continue
		}
		rules = append(rules,
			validate.Scoped(validate.StepMonotonic(shape.Slot, shape.YearColumn), shape.Slot),
			validate.Scoped(validate.StepCoversStart(shape.Slot, shape.YearColumn, planStart), shape.Slot),
		)
	}

	return rules
}

type ColumnPair struct {
	Slot             tsv.Slot
	Positive, Needed tsv.ColumnName

	Why string
}

func PositiveTogether() []ColumnPair {
	var pairs []ColumnPair

	for _, slot := range []tsv.Slot{IncomeHusbandSlot, IncomeWifeSlot} {
		for _, paid := range []tsv.ColumnName{AnnualSalaryColumn, BonusColumn} {
			pairs = append(pairs, ColumnPair{
				Slot: slot, Positive: paid, Needed: WeeklyHoursColumn,
				Why: "給与や賞与を受け取る年は働いている年である。週所定労働時間が 0 のままだと被用者保険に入らず、" +
					"健康保険料・厚生年金保険料・雇用保険料のどれも引かれない",
			})
		}
	}
	return pairs
}
