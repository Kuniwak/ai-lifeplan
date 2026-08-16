package plan_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

const repoRoot = "../"

func TestLoadShouldStartFromTheRecordTheProjectNames(t *testing.T) {
	full, err := tsv.ReadFile(filepath.Join(repoRoot, "actuals", "balance.tsv"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(full.Rows) < 2 {
		t.Fatalf("%d row(s) of record, want at least 2 to cut", len(full.Rows))
	}

	cut := filepath.Join(t.TempDir(), "balance-cut.tsv")
	if err := tsv.WriteFile(cut, &tsv.Table{Header: full.Header, Rows: full.Rows[:1]}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	latest, err := plan.Load(plan.Sources{
		Root: repoRoot, ProjectPath: filepath.Join(repoRoot, "projects", "base.tsv"),
	})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	earlier, err := plan.Load(plan.Sources{
		Root:          repoRoot,
		ProjectPath:   filepath.Join(repoRoot, "projects", "base.tsv"),
		SlotOverrides: map[tsv.Slot]string{input.BalanceSlot: cut},
	})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}

	was, err := latest.StartsAfter()
	if err != nil {
		t.Fatalf("StartsAfter: %v", err)
	}
	is, err := earlier.StartsAfter()
	if err != nil {
		t.Fatalf("StartsAfter: %v", err)
	}
	if is >= was {
		t.Errorf("起点が %d のままである（%d より前になるはず）", is, was)
	}
}

func TestLoadShouldRefuseAProjectWithNoRecord(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "no-balance.tsv")
	base, err := os.ReadFile(filepath.Join(repoRoot, "projects", "base.tsv"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var kept []string
	for _, line := range strings.Split(string(base), "\n") {
		if strings.HasPrefix(line, string(input.BalanceSlot)+"\t") {
			continue
		}
		kept = append(kept, line)
	}
	if err := os.WriteFile(manifest, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = plan.Load(plan.Sources{Root: repoRoot, ProjectPath: manifest})

	if err == nil {
		t.Fatal("plan.Load accepted a project with no record to start from")
	}
	if !strings.Contains(err.Error(), string(input.BalanceSlot)) {
		t.Errorf("%q does not name the slot", err)
	}
}

func TestBuildShouldRefuseInflationInAYearTheRecordCovers(t *testing.T) {
	moved := filepath.Join(t.TempDir(), "inflation-moved.tsv")
	if err := os.WriteFile(moved, []byte("西暦\tインフレ率\n2018\t0.00%\n2023\t3.20%\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	in, err := plan.Load(plan.Sources{
		Root:          repoRoot,
		ProjectPath:   filepath.Join(repoRoot, "projects", "base.tsv"),
		SlotOverrides: map[tsv.Slot]string{input.InflationSlot: moved},
	})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}

	_, err = in.Build()

	if err == nil {
		t.Fatal("実績のある年に物価が動く計画を受け付けている")
	}
	for _, want := range []string{"2023", "物価", "inflation.tsv"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%q が %q を言っていない", err, want)
		}
	}
}
