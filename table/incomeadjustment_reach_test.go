package table_test

import (
	"path/filepath"
	"testing"

	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func forEachIncomeTableOfEveryProject(t *testing.T, f func(project string, person table.PersonName, income relation.Table[table.IncomeRow])) {
	t.Helper()

	manifests, err := filepath.Glob("../projects/*.tsv")
	if err != nil {
		t.Fatalf("filepath.Glob: %v", err)
	}
	more, err := filepath.Glob("../projects/counterfactual/*.tsv")
	if err != nil {
		t.Fatalf("filepath.Glob: %v", err)
	}
	manifests = append(manifests, more...)
	if len(manifests) == 0 {
		t.Fatal("projects/ に木が 1 つも無い")
	}

	for _, manifest := range manifests {
		built, err := plan.Build(plan.Sources{Root: "..", ProjectPath: manifest})
		if err != nil {
			t.Fatalf("plan.Build(%s): %v", manifest, err)
		}
		for person, income := range built.Income {
			f(filepath.Base(manifest), person, income)
		}
	}
}
