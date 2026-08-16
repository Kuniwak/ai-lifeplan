package validate

import (
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/sets"
	"github.com/Kuniwak/lifeplan/tsv"
)

func genSlots() *rapid.Generator[[]tsv.Slot] {
	return rapid.SliceOfNDistinct(
		rapid.SampledFrom([]tsv.Slot{"household", "market", "income_wife", "living_cost", "balance"}),
		0, 5,
		func(s tsv.Slot) tsv.Slot { return s },
	)
}

func genRules() *rapid.Generator[[]Rule] {
	return rapid.Custom(func(t *rapid.T) []Rule {
		names := rapid.SliceOfNDistinct(
			rapid.SampledFrom([]RuleName{"r1", "r2", "r3", "r4"}),
			0, 4,
			func(s RuleName) RuleName { return s },
		).Draw(t, "names")

		rules := make([]Rule, 0, len(names))
		for _, name := range names {
			needs := genSlots().Draw(t, "needs-"+string(name))
			if len(needs) == 0 {
				needs = []tsv.Slot{"household"}
			}
			rules = append(rules, Rule{
				Name:  name,
				Needs: needs,
				Check: func(map[tsv.Slot]*tsv.Table) []Finding { return nil },
			})
		}
		return rules
	})
}

func presentTables(slots []tsv.Slot) map[tsv.Slot]*tsv.Table {
	present := make(map[tsv.Slot]*tsv.Table, len(slots))
	for _, slot := range slots {
		present[slot] = &tsv.Table{Header: []tsv.ColumnName{"西暦"}}
	}
	return present
}

func TestRequireAllShouldNeverPassWhenARuleCouldNotBeDecided(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rules := genRules().Draw(t, "rules")
		present := presentTables(genSlots().Draw(t, "present"))

		undecidable := false
		for _, rule := range rules {
			if len(absentSlots(rule, present)) > 0 {
				undecidable = true
			}
		}

		got := Run(rules, present, RequireAll)

		if undecidable && got.OK() {
			t.Fatalf("a rule could not be decided yet the run passed: %+v", got)
		}
		if !undecidable && !got.OK() {
			t.Fatalf("every rule could be decided and none complained, yet the run failed: %+v", got)
		}
	})
}

func TestAllowMissingShouldSkipExactlyTheUndecidableRules(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rules := genRules().Draw(t, "rules")
		present := presentTables(genSlots().Draw(t, "present"))

		var wantSkipped []RuleName
		for _, rule := range rules {
			if len(absentSlots(rule, present)) > 0 {
				wantSkipped = append(wantSkipped, rule.Name)
			}
		}
		slices.Sort(wantSkipped)

		got := Run(rules, present, AllowMissing)

		if extra := sets.Difference(got.Skipped, wantSkipped); len(extra) > 0 {
			t.Fatalf("rules were skipped although their tables were there: %v", extra)
		}
		if missed := sets.Difference(wantSkipped, got.Skipped); len(missed) > 0 {
			t.Fatalf("rules ran although a table they need was absent: %v", missed)
		}
	})
}

func TestEveryRuleShouldEitherRunOrBeAccountedFor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rules := genRules().Draw(t, "rules")
		present := presentTables(genSlots().Draw(t, "present"))
		missing := rapid.SampledFrom([]Missing{RequireAll, AllowMissing}).Draw(t, "missing")

		var names []RuleName
		for _, rule := range rules {
			names = append(names, rule.Name)
		}

		got := Run(rules, present, missing)

		accounted := slices.Concat(got.Ran, got.Skipped)
		for _, f := range got.Findings {
			accounted = append(accounted, f.Rule)
		}
		if lost := sets.Difference(names, accounted); len(lost) > 0 {
			t.Fatalf("rules disappeared without being run, skipped or reported: %v", lost)
		}
	})
}

func TestAPartialRunShouldNeverBeIndistinguishableFromACompleteOne(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rules := genRules().Draw(t, "rules")
		present := presentTables(genSlots().Draw(t, "present"))

		got := Run(rules, present, AllowMissing)

		if len(got.Skipped) > 0 && !got.Partial() {
			t.Fatalf("rules were skipped but the result does not say so: %+v", got)
		}
		if len(got.Skipped) == 0 && got.Partial() {
			t.Fatalf("nothing was skipped but the result claims to be partial: %+v", got)
		}
	})
}
