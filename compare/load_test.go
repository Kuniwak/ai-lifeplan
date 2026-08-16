package compare_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/tsv"
)

const repoRoot = "../"

func manifests(names ...string) []string {
	paths := make([]string, len(names))
	for i, name := range names {
		paths[i] = filepath.Join(repoRoot, "projects", name+".tsv")
	}
	return paths
}

func TestLoadShouldNameEachProjectAfterItsManifest(t *testing.T) {
	got, err := compare.Load(compare.Sources{
		Root: repoRoot, ProjectPaths: manifests("base", "settle-2050"),
	})

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("%d subject(s), want 2", len(got))
	}
	for i, want := range []string{"base", "settle-2050"} {
		if got[i].Name != want {
			t.Errorf("subject %d is named %q, want %q", i, got[i].Name, want)
		}
	}
	if got[0].StartsAfter == 0 {
		t.Error("the starting year was not read")
	}
	if len(got[0].Tables) == 0 {
		t.Error("no tables were worked out")
	}
	if len(got[0].Paths) == 0 {
		t.Error("no input paths were resolved")
	}
}

func TestLoadShouldRefuseTwoProjectsOfTheSameName(t *testing.T) {
	_, err := compare.Load(compare.Sources{
		Root: repoRoot, ProjectPaths: manifests("base", "base"),
	})

	if err == nil {
		t.Fatal("Load accepted the same project twice")
	}
	if !strings.Contains(err.Error(), "base") {
		t.Errorf("%q does not name the project", err)
	}
}

func TestLoadShouldRefuseFewerThanTwoProjects(t *testing.T) {
	_, err := compare.Load(compare.Sources{
		Root: repoRoot, ProjectPaths: manifests("base"),
	})

	if err == nil {
		t.Fatal("Load accepted a single project")
	}
}

func TestOfShouldProduceEveryTableOfAComparison(t *testing.T) {
	subjects := []compare.Subject{
		{
			Name: "base", StartsAfter: 2029,
			Paths: map[tsv.Slot]string{"inflation": "data/environment/inflation.tsv"},
			Tables: twoYears(
				[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
				[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"}),
		},
		{
			Name: "zero-growth", StartsAfter: 2029,
			Paths: map[tsv.Slot]string{"inflation": "data/environment/scenario/inflation-zero.tsv"},
			Tables: twoYears(
				[2]string{"100", "110"}, [2]string{"80", "900"}, [2]string{"20", "-790"},
				[2]string{"1000", "0"}, [2]string{"0", "60"}, [2]string{"9", "9"}),
		},
	}

	got, err := compare.Of(subjects)

	if err != nil {
		t.Fatalf("Of: %v", err)
	}
	for _, want := range []compare.OutputName{
		compare.SlotsOutput, compare.TimelineOutput, compare.DiffOutput, compare.SummaryOutput,
	} {
		table, ok := got.Tables()[want]
		if !ok {
			t.Errorf("no %s table", want)
			continue
		}
		if len(table.Rows) == 0 {
			t.Errorf("%s has no rows", want)
		}
	}
}

func TestLoadShouldRefuseAProjectNamedAfterTheRecord(t *testing.T) {
	_, err := compare.Load(compare.Sources{
		Root:         ".",
		ProjectPaths: []string{"projects/実績.tsv", "projects/base.tsv"},
	})

	if err == nil {
		t.Fatal("Load accepted a project named after the record")
	}
	if !strings.Contains(err.Error(), compare.RecordSeries) {
		t.Errorf("%q が %q を名指ししていない", err, compare.RecordSeries)
	}
}

func TestLoadShouldApplyAnOverrideToEveryProject(t *testing.T) {
	got, err := compare.Load(compare.Sources{
		Root:         repoRoot,
		ProjectPaths: manifests("base", "settle-2050"),
		SlotOverrides: map[tsv.Slot]string{
			"inflation": "data/environment/scenario/inflation-2percent.tsv",
		},
	})

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, subject := range got {
		if subject.Paths["inflation"] != "data/environment/scenario/inflation-2percent.tsv" {
			t.Errorf("%s の inflation = %q, 上書きが効いていない",
				subject.Name, subject.Paths["inflation"])
		}
	}
}

func TestLoadShouldRefuseAnOverrideOfASlotNothingSets(t *testing.T) {
	_, err := compare.Load(compare.Sources{
		Root:          repoRoot,
		ProjectPaths:  manifests("base", "settle-2050"),
		SlotOverrides: map[tsv.Slot]string{"inflatoin": "x.tsv"},
	})

	if err == nil {
		t.Fatal("Load accepted an override of a slot nothing sets")
	}
	if !strings.Contains(err.Error(), "inflatoin") {
		t.Errorf("%q does not name the slot", err)
	}
}
