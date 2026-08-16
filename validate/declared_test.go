package validate

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestDeclaredRulesShouldNameEveryRuleThisPackageDefines(t *testing.T) {
	got := DeclaredRules()

	for _, want := range []RuleName{
		AscendingRule, ColumnSchemaRule, StepMonotonicRule, StepCoversStartRule,
		YearGapRule, ValueRangeRule, YearCoverageRule, SlotResolvableRule,
		LawSourceRule, LawValidityRule, LawRangeTotalRule, MunicipalityRule,
	} {
		if !slices.Contains(got, want) {
			t.Errorf("DeclaredRules() does not name %q", want)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("DeclaredRules() is not in order: %v", got)
	}
}

func TestUnregisteredShouldNameARuleNobodyWiredUp(t *testing.T) {
	registry := NewRegistry([]Rule{ColumnSchema("s", nil)})

	got := registry.Unregistered()

	named := make(map[RuleName]string, len(got))
	for _, d := range got {
		named[d.Name] = d.Unwired
	}
	if _, ok := named[ColumnSchemaRule]; ok {
		t.Errorf("Unregistered() names a rule that is registered: %v", got)
	}
	if _, ok := named[YearGapRule]; !ok {
		t.Errorf("Unregistered() does not name %q, which nothing wired up: %v", YearGapRule, got)
	}
	if reason := named[AscendingRule]; reason != "" {
		t.Errorf("ascending に配線しない理由が書いてある: %q", reason)
	}
	if reason := named[YearGapRule]; reason == "" {
		t.Errorf("year-gap を配線しない理由が書かれていない")
	}
}

func TestUnregisteredShouldSayNothingWhenEveryRuleIsWiredUp(t *testing.T) {
	rules := make([]Rule, 0, len(DeclaredRules()))
	for _, name := range DeclaredRules() {
		rules = append(rules, Rule{Name: name, Needs: []tsv.Slot{}})
	}
	registry := NewRegistry(rules)

	if got := registry.Unregistered(); len(got) != 0 {
		t.Errorf("Unregistered() = %v, want none", got)
	}
}

func TestEveryUnwiredRuleShouldSayWhy(t *testing.T) {
	for _, d := range Declarations() {
		if d.Unwired == "" {
			continue
		}
		if len([]rune(d.Unwired)) < 10 {
			t.Errorf("%q の理由 %q が短すぎる。いつ配線するのか、あるいはなぜ配線しないのかが読めない", d.Name, d.Unwired)
		}
	}
}

func TestUnwiredWithoutAReasonShouldNameARuleNobodyExplained(t *testing.T) {
	registry := NewRegistry([]Rule{ColumnSchema("s", nil)})

	var unexplained, onPurpose []RuleName
	for _, d := range registry.UnwiredWithoutAReason() {
		unexplained = append(unexplained, d.Name)
	}
	for _, d := range registry.UnwiredOnPurpose() {
		onPurpose = append(onPurpose, d.Name)
	}

	if !slices.Contains(unexplained, AscendingRule) {
		t.Errorf("理由の無い %q が「理由なく配線されていない」に出ていない: %v", AscendingRule, unexplained)
	}
	if slices.Contains(unexplained, YearGapRule) {
		t.Errorf("理由のある %q が「理由なく配線されていない」に出ている: %v", YearGapRule, unexplained)
	}
	if !slices.Contains(onPurpose, YearGapRule) {
		t.Errorf("理由のある %q が「意図して配線していない」に出ていない: %v", YearGapRule, onPurpose)
	}
	if slices.Contains(onPurpose, AscendingRule) {
		t.Errorf("理由の無い %q が「意図して配線していない」に出ている: %v", AscendingRule, onPurpose)
	}
	if got, want := len(unexplained)+len(onPurpose), len(registry.Unregistered()); got != want {
		t.Errorf("二つに分けた合計が %d 件、Unregistered() は %d 件。どちらにも入らない規則がある", got, want)
	}
}

func TestWiredWithAReasonShouldCatchAStaleReason(t *testing.T) {
	registry := NewRegistry([]Rule{{Name: YearGapRule, Needs: []tsv.Slot{}}})

	got := registry.WiredWithAReason()
	if len(got) != 1 || got[0].Name != YearGapRule {
		t.Fatalf("WiredWithAReason() = %v, want just %q", got, YearGapRule)
	}

	if stale := NewRegistry(nil).WiredWithAReason(); len(stale) != 0 {
		t.Errorf("WiredWithAReason() = %v, want none", stale)
	}
}
