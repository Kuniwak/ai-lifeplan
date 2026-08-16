package validate

import (
	"slices"
	"strings"
)

type Declaration struct {
	Name RuleName

	Unwired string
}

func DeclaredRules() []RuleName {
	declared := Declarations()
	names := make([]RuleName, 0, len(declared))
	for _, d := range declared {
		names = append(names, d.Name)
	}
	slices.Sort(names)
	return names
}

func Declarations() []Declaration {
	declared := []Declaration{
		{Name: AmountAgreesWithItsStateRule},
		{Name: StateOnlyAtTheStartRule},
		{Name: AmountsAddUpRule},
		{Name: AscendingRule},
		{Name: BalanceFollowsTheBankRule},
		{Name: BalanceFollowsTheKnownRule},
		{Name: BalanceFollowsStatementsRule},
		{Name: OutsideFollowsTheHoldingsRule},
		{Name: ColumnSchemaRule},
		{Name: EveryChoiceMadeRule},
		{Name: HoldingsFollowLedgerRule},
		{Name: ItemsKnownRule},
		{Name: KeysAreDeclaredRule},
		{Name: KeysCoverYearsRule},
		{Name: LawRangeTotalRule},
		{Name: LawSourceRule},
		{Name: LawValidityRule},
		{Name: LastMunicipalityRule},
		{Name: LoanFixedPeriodRule},
		{Name: LoanSettlementInTermRule},
		{Name: MunicipalityRule},
		{Name: PositiveNeedsRule},
		{Name: SlotResolvableRule},
		{Name: StatementsCoverRule},
		{Name: StateOnlyAtTheEndRule},
		{Name: StepCoversStartRule},
		{Name: StepMonotonicRule},
		{Name: UnclassifiedRule},
		{Name: UniqueKeyRule},

		{Name: ValueRangeRule, Unwired: "使い手がいない。範囲は列の Parser が持つようになった。残すか消すかは未決"},

		{Name: YearCoverageRule, Unwired: "どの表とどの表が同じ年を覆うべきかを、まだ誰も書いていない"},
		{Name: YearsAreCoveredRule},
		{Name: YearGapRule, Unwired: "年の欠落と重複は unique-key と step-covers-start が表ごとに見ており、この規則を足すと同じことを二度言う"},
		{Name: YearsOutsideComparisonRule},
	}
	return declared
}

func (r Registry) Unregistered() []Declaration {
	registered := r.registeredNames()

	var missing []Declaration
	for _, d := range Declarations() {
		if _, ok := registered[d.Name]; !ok {
			missing = append(missing, d)
		}
	}
	slices.SortFunc(missing, compareDeclarations)
	return missing
}

func (r Registry) UnwiredWithoutAReason() []Declaration {
	return filterDeclarations(r.Unregistered(), func(d Declaration) bool { return d.Unwired == "" })
}

func (r Registry) UnwiredOnPurpose() []Declaration {
	return filterDeclarations(r.Unregistered(), func(d Declaration) bool { return d.Unwired != "" })
}

func (r Registry) WiredWithAReason() []Declaration {
	registered := r.registeredNames()

	var stale []Declaration
	for _, d := range Declarations() {
		if d.Unwired == "" {
			continue
		}
		if _, ok := registered[d.Name]; ok {
			stale = append(stale, d)
		}
	}
	slices.SortFunc(stale, compareDeclarations)
	return stale
}

func compareDeclarations(a, b Declaration) int {
	return strings.Compare(string(a.Name), string(b.Name))
}

func filterDeclarations(all []Declaration, keep func(Declaration) bool) []Declaration {
	var kept []Declaration
	for _, d := range all {
		if keep(d) {
			kept = append(kept, d)
		}
	}
	return kept
}

func (r Registry) registeredNames() map[RuleName]struct{} {
	registered := make(map[RuleName]struct{}, len(r.rules))
	for _, rule := range r.rules {
		name, _, _ := strings.Cut(string(rule.Name), ScopeSeparator)
		registered[RuleName(name)] = struct{}{}
	}
	return registered
}
