package main

import (
	"testing"
)

const repoRoot = "../.."

func TestEveryAxisShouldStartAtTheBase(t *testing.T) {
	for _, a := range Axes() {
		if len(a.levels) < 2 {
			t.Errorf("軸 %q の水準が %d しかない", a.name, len(a.levels))
		}
		if got := a.levels[0].slots; len(got) != 0 {
			t.Errorf("軸 %q の最初の水準 %q が %d 個の slot を上書きしている: %v",
				a.name, a.levels[0].key, len(got), got)
		}
	}
}

func TestCellsShouldBeTheProductOfEveryAxis(t *testing.T) {
	axes := Axes()
	want := 1
	for _, a := range axes {
		want *= len(a.levels)
	}
	if got := len(Cells(axes)); got != want {
		t.Errorf("%d セル、欲しいのは %d", got, want)
	}

	seen := make(map[string]bool)
	for _, cell := range Cells(axes) {
		key := ""
		for i, a := range axes {
			key += a.levels[cell[i]].key + "\t"
		}
		if seen[key] {
			t.Fatalf("同じセルが二度ある: %q", key)
		}
		seen[key] = true
	}
}
