package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

type RuleName string

type Finding struct {
	Rule RuleName

	Slot tsv.Slot

	Message string
}

func (f Finding) String() string {
	if f.Slot == "" {
		return fmt.Sprintf("%s: %s", f.Rule, f.Message)
	}
	return fmt.Sprintf("%s: %s: %s", f.Rule, f.Slot, f.Message)
}

type Rule struct {
	Name RuleName

	Needs []tsv.Slot

	Check func(tables map[tsv.Slot]*tsv.Table) []Finding
}

const ScopeSeparator = "/"

func Scoped(rule Rule, slot tsv.Slot) Rule {
	rule.Name += RuleName(ScopeSeparator + string(slot))
	return rule
}

type Registry struct {
	rules []Rule
}

func NewRegistry(rules []Rule) Registry {
	sorted := slices.Clone(rules)
	slices.SortFunc(sorted, func(a, b Rule) int { return strings.Compare(string(a.Name), string(b.Name)) })

	for i, rule := range sorted {
		if rule.Needs == nil {
			panic(fmt.Sprintf("validate: rule %q does not declare what it needs; write an empty list to say it needs no table", rule.Name))
		}
		if i > 0 && rule.Name == sorted[i-1].Name {
			panic(fmt.Sprintf("validate: rule %q is registered twice", rule.Name))
		}
	}

	return Registry{rules: sorted}
}

func (r Registry) Rules() []Rule {
	return slices.Clone(r.rules)
}

type Missing int

const (
	RequireAll Missing = iota

	AllowMissing
)

type Result struct {
	Ran []RuleName

	Skipped []RuleName

	Findings []Finding
}

func (r Result) OK() bool {
	return len(r.Findings) == 0
}

func (r Result) Partial() bool {
	return len(r.Skipped) > 0
}

func Run(rules []Rule, present map[tsv.Slot]*tsv.Table, missing Missing) Result {
	registry := NewRegistry(rules)

	var result Result
	for _, rule := range registry.Rules() {
		absent := absentSlots(rule, present)

		if len(absent) > 0 {
			if missing == AllowMissing {
				result.Skipped = append(result.Skipped, rule.Name)
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Rule: rule.Name,
				Slot: absent[0],
				Message: fmt.Sprintf(
					"cannot be checked: the input table(s) %s are not there",
					joinSlots(absent, ", ")),
			})
			continue
		}

		result.Ran = append(result.Ran, rule.Name)
		for _, finding := range rule.Check(neededTables(rule, present)) {
			finding.Rule = rule.Name
			result.Findings = append(result.Findings, finding)
		}
	}

	return result
}

func absentSlots(rule Rule, present map[tsv.Slot]*tsv.Table) []tsv.Slot {
	var absent []tsv.Slot
	for _, slot := range rule.Needs {
		if _, ok := present[slot]; !ok {
			absent = append(absent, slot)
		}
	}
	slices.Sort(absent)
	return absent
}

func neededTables(rule Rule, present map[tsv.Slot]*tsv.Table) map[tsv.Slot]*tsv.Table {
	needed := make(map[tsv.Slot]*tsv.Table, len(rule.Needs))
	for _, slot := range rule.Needs {
		needed[slot] = present[slot]
	}
	return needed
}
