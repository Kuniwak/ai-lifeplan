package validate

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/tsv"
)

func alwaysPasses(name RuleName, needs ...tsv.Slot) Rule {
	return Rule{
		Name:  name,
		Needs: needs,
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding { return nil },
	}
}

func alwaysFails(name RuleName, count int, needs ...tsv.Slot) Rule {
	return Rule{
		Name:  name,
		Needs: needs,
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			var found []Finding
			for i := range count {
				found = append(found, Finding{
					Slot:    needs[0],
					Message: fmt.Sprintf("%s complaint %d", name, i+1),
				})
			}
			return found
		},
	}
}

func table() *tsv.Table {
	return &tsv.Table{Header: []tsv.ColumnName{"西暦"}, Rows: [][]string{{"2031"}}}
}

func TestRunShouldOnlyRunTheRulesWhoseTablesArePresent(t *testing.T) {
	rules := []Rule{
		alwaysPasses("within-household", "household"),
		alwaysPasses("within-market", "market"),
		alwaysPasses("across-both", "household", "market"),
	}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, AllowMissing)

	if diff := cmp.Diff([]RuleName{"within-household"}, got.Ran); diff != "" {
		t.Errorf("Ran mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]RuleName{"across-both", "within-market"}, got.Skipped); diff != "" {
		t.Errorf("Skipped mismatch (-want +got):\n%s", diff)
	}
}

func TestRunShouldReportEveryFindingAtOnce(t *testing.T) {
	rules := []Rule{
		alwaysFails("rule-a", 2, "household"),
		alwaysFails("rule-b", 3, "household"),
	}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, RequireAll)

	if len(got.Findings) != 5 {
		t.Fatalf("want all 5 findings, got %d: %v", len(got.Findings), got.Findings)
	}
	if got.OK() {
		t.Error("OK() reported success although rules complained")
	}
}

func TestRunShouldNameTheRuleOnEveryFinding(t *testing.T) {
	rules := []Rule{alwaysFails("year-gap", 1, "household")}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, RequireAll)

	if len(got.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got.Findings))
	}
	if got.Findings[0].Rule != "year-gap" {
		t.Errorf("Rule = %q, want the rule that produced it", got.Findings[0].Rule)
	}
}

func TestRunRequireAllShouldFailWhenATableIsMissing(t *testing.T) {
	rules := []Rule{alwaysPasses("across-both", "household", "market")}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, RequireAll)

	if got.OK() {
		t.Error("OK() reported success although a rule could not be checked")
	}
	if len(got.Findings) != 1 {
		t.Fatalf("want a finding about the missing table, got %v", got.Findings)
	}
	if !strings.Contains(got.Findings[0].Message, "market") {
		t.Errorf("the finding does not name the missing table: %q", got.Findings[0].Message)
	}
}

func TestRunAllowMissingShouldSucceedButSayWhatItSkipped(t *testing.T) {
	rules := []Rule{
		alwaysPasses("within-household", "household"),
		alwaysPasses("across-both", "household", "market"),
	}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, AllowMissing)

	if !got.OK() {
		t.Errorf("OK() reported failure although every runnable rule passed: %v", got.Findings)
	}
	if diff := cmp.Diff([]RuleName{"across-both"}, got.Skipped); diff != "" {
		t.Errorf("Skipped mismatch (-want +got):\n%s", diff)
	}
	if !got.Partial() {
		t.Error("Partial() reported a complete check although a rule was skipped")
	}
}

func TestRunShouldNotBePartialWhenNothingWasSkipped(t *testing.T) {
	rules := []Rule{alwaysPasses("within-household", "household")}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, AllowMissing)

	if got.Partial() {
		t.Error("Partial() reported a partial check although every rule ran")
	}
}

func TestRunShouldHandOnlyTheNeededTablesToARule(t *testing.T) {
	var handed []tsv.Slot
	rules := []Rule{{
		Name:  "within-household",
		Needs: []tsv.Slot{"household"},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			for slot := range tables {
				handed = append(handed, slot)
			}
			return nil
		},
	}}
	present := map[tsv.Slot]*tsv.Table{"household": table(), "market": table()}

	Run(rules, present, AllowMissing)

	slices.Sort(handed)
	if diff := cmp.Diff([]tsv.Slot{"household"}, handed); diff != "" {
		t.Errorf("a rule was handed tables it did not ask for (-want +got):\n%s", diff)
	}
}

func TestRunShouldOrderTheResultsByName(t *testing.T) {
	rules := []Rule{
		alwaysPasses("zzz", "household"),
		alwaysPasses("aaa", "household"),
		alwaysPasses("mmm", "household"),
	}
	present := map[tsv.Slot]*tsv.Table{"household": table()}

	got := Run(rules, present, RequireAll)

	if diff := cmp.Diff([]RuleName{"aaa", "mmm", "zzz"}, got.Ran); diff != "" {
		t.Errorf("Ran is not ordered (-want +got):\n%s", diff)
	}
}

func TestRunWithNoRules(t *testing.T) {
	got := Run(nil, map[tsv.Slot]*tsv.Table{}, RequireAll)

	if !got.OK() {
		t.Error("OK() reported failure with nothing to check")
	}
	if got.Partial() {
		t.Error("Partial() reported a partial check with nothing to check")
	}
}

func TestNewRegistryShouldPanicOnADuplicatedRuleName(t *testing.T) {
	refused := panictest.Recovered(func() {
		NewRegistry([]Rule{alwaysPasses("same", "a"), alwaysPasses("same", "b")})
	})

	if refused == nil {
		t.Error("want panic for two rules sharing a name, got none")
	}
}

func TestNewRegistryShouldPanicOnARuleThatNeverDeclaredWhatItNeeds(t *testing.T) {
	refused := panictest.Recovered(func() {
		NewRegistry([]Rule{{Name: "undeclared", Check: func(map[tsv.Slot]*tsv.Table) []Finding { return nil }}})
	})

	if refused == nil {
		t.Error("want panic for a rule that left Needs nil, got none")
	}
}

func TestARuleThatNeedsNoTableShouldAlwaysRun(t *testing.T) {
	rule := Rule{
		Name:  "about-the-manifest",
		Needs: []tsv.Slot{},
		Check: func(map[tsv.Slot]*tsv.Table) []Finding { return nil },
	}

	for name, missing := range map[string]Missing{"RequireAll": RequireAll, "AllowMissing": AllowMissing} {
		t.Run(name, func(t *testing.T) {
			got := Run([]Rule{rule}, map[tsv.Slot]*tsv.Table{}, missing)

			if diff := cmp.Diff([]RuleName{"about-the-manifest"}, got.Ran); diff != "" {
				t.Errorf("the rule did not run (-want +got):\n%s", diff)
			}
			if got.Partial() {
				t.Error("Partial() reported a partial check although the rule needs no table")
			}
		})
	}
}

func TestRegistryRulesShouldBeOrderedByName(t *testing.T) {
	registry := NewRegistry([]Rule{
		alwaysPasses("zzz", "a"),
		alwaysPasses("aaa", "a"),
	})

	var names []RuleName
	for _, r := range registry.Rules() {
		names = append(names, r.Name)
	}

	if diff := cmp.Diff([]RuleName{"aaa", "zzz"}, names); diff != "" {
		t.Errorf("Rules is not ordered (-want +got):\n%s", diff)
	}
}

func TestScopedShouldNameTheRuleAfterTheTableItIsAbout(t *testing.T) {
	scoped := Scoped(ColumnSchema("income_husband", nil), "income_husband")

	if got, want := scoped.Name, ColumnSchemaRule+"/income_husband"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
}

func TestRunShouldDecideOneRuleOverSeveralTables(t *testing.T) {
	rules := []Rule{
		Scoped(ColumnSchema("income_husband", nil), "income_husband"),
		Scoped(ColumnSchema("income_wife", nil), "income_wife"),
	}
	present := map[tsv.Slot]*tsv.Table{
		"income_husband": {Header: []tsv.ColumnName{"西暦"}},
	}

	got := Run(rules, present, AllowMissing)

	if want := []RuleName{ColumnSchemaRule + "/income_husband"}; !slices.Equal(got.Ran, want) {
		t.Errorf("Ran = %v, want %v", got.Ran, want)
	}
	if want := []RuleName{ColumnSchemaRule + "/income_wife"}; !slices.Equal(got.Skipped, want) {
		t.Errorf("Skipped = %v, want %v", got.Skipped, want)
	}
}

func TestAFindingAboutMissingTablesShouldNameThemAllSeparately(t *testing.T) {
	rules := []Rule{alwaysPasses("needs-two", "household", "market")}

	got := Run(rules, map[tsv.Slot]*tsv.Table{}, RequireAll)

	if len(got.Findings) != 1 {
		t.Fatalf("findings が %d 件ある（1 件のはず）: %v", len(got.Findings), got.Findings)
	}
	if want := "the input table(s) household, market are not there"; !strings.Contains(got.Findings[0].Message, want) {
		t.Errorf("%q と言っていない: %q", want, got.Findings[0].Message)
	}
}
