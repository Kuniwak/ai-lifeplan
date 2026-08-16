package plan_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestInputPathsShouldNameWhatNoManifestNames(t *testing.T) {
	sources := plan.Sources{Root: "..", ProjectPath: "../projects/classic.tsv"}

	paths, err := plan.InputPaths(sources)

	if err != nil {
		t.Fatalf("InputPaths: %v", err)
	}
	for _, want := range []string{
		filepath.Join("..", "data", "law", "national", "employment-insurance-rate.tsv"),
		filepath.Join("..", "actuals", "cashflow.tsv"),
		filepath.Join("..", "actuals", "adjustments.tsv"),
		filepath.Join("..", "projects", "base.tsv"),
		filepath.Join("..", "data", "controllable", "income-husband.tsv"),
	} {
		if !slices.Contains(paths, want) {
			t.Errorf("want %s among the files the plan is read from", want)
		}
	}
}

func TestInputPathsShouldNameEachFileOnce(t *testing.T) {
	sources := plan.Sources{Root: "..", ProjectPath: "../projects/case-high-growth.tsv"}

	paths, err := plan.InputPaths(sources)

	if err != nil {
		t.Fatalf("InputPaths: %v", err)
	}
	if !slices.IsSorted(paths) {
		t.Errorf("want the paths sorted, so that a build file sees the same list every time")
	}
	if len(slices.Compact(slices.Clone(paths))) != len(paths) {
		t.Errorf("want no path twice, got %d paths with repeats", len(paths))
	}
}

func TestInputPathsShouldNameEveryTableTheChecksRead(t *testing.T) {
	sources := plan.Sources{Root: "..", ProjectPath: "../projects/classic.tsv"}

	paths, err := plan.InputPaths(sources)
	if err != nil {
		t.Fatalf("InputPaths: %v", err)
	}
	_, checked, err := actuals.Rules("..")
	if err != nil {
		t.Fatalf("actuals.Rules: %v", err)
	}

	for slot := range checked {
		want := tsv.Under("..", string(slot))
		if !slices.Contains(paths, want) {
			t.Errorf("want %s among the files the plan is read from: the checks read it, so a change to it must work the plan out again", want)
		}
	}
	if want := tsv.Under("..", string(actuals.AccountsPath)); !slices.Contains(paths, want) {
		t.Errorf("want %s among the files the plan is read from", want)
	}
}
