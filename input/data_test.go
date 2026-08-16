package input_test

import (
	"github.com/Kuniwak/lifeplan/money"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

const repoRoot = ".."

const baseManifest = "../projects/base.tsv"

func TestTheBaseProjectShouldFillEverySlotThePlanNeeds(t *testing.T) {
	p, err := project.Load(baseManifest)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}

	filled := p.SlotNames()
	needed := input.RequiredSlots()

	for _, slot := range needed {
		if !slices.Contains(filled, slot) {
			t.Errorf("the base project fills no table for the slot %q", slot)
		}
	}
	for _, slot := range filled {
		if !slices.Contains(needed, slot) {
			t.Errorf("the base project fills the slot %q, which the plan does not use", slot)
		}
	}
}

func TestTheBaseProjectShouldPointAtTablesThatParse(t *testing.T) {
	p, err := project.Load(baseManifest)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}

	paths := make(map[tsv.Slot]string, len(p.SlotNames()))
	for _, slot := range p.SlotNames() {
		path, _ := p.Path(slot)
		paths[slot] = path
	}

	tables, err := input.Load(repoRoot, paths)
	if err != nil {
		t.Fatalf("input.Load: %v", err)
	}

	for _, slot := range input.RequiredSlots() {
		if _, ok := tables[slot]; !ok {
			path := filepath.Join(repoRoot, filepath.FromSlash(paths[slot]))
			t.Errorf("the slot %q was not read; %s is not there", slot, path)
		}
	}
}

func loadBase(t *testing.T) map[tsv.Slot]*tsv.Table {
	t.Helper()

	p, err := project.Load(baseManifest)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}

	paths := make(map[tsv.Slot]string, len(p.SlotNames()))
	for _, slot := range p.SlotNames() {
		path, _ := p.Path(slot)
		paths[slot] = path
	}

	tables, err := input.Load(repoRoot, paths)
	if err != nil {
		t.Fatalf("input.Load: %v", err)
	}
	return tables
}

func TestThePlanTableShouldStateTheSpanTheOtherTablesAreReadOver(t *testing.T) {
	from, to, err := input.PlanSpan(loadBase(t)[input.PlanSlot])
	if err != nil {
		t.Fatalf("input.PlanSpan: %v", err)
	}

	if from != planStart || to != planEnd {
		t.Errorf("span = %d..%d, want %d..%d", from, to, planStart, planEnd)
	}
}

func TestTheBaseProjectShouldSatisfyEveryInvariantOfTheInput(t *testing.T) {
	tables := loadBase(t)

	from, _, err := input.PlanSpan(tables[input.PlanSlot])
	if err != nil {
		t.Fatalf("input.PlanSpan: %v", err)
	}

	got := validate.Run(input.Rules(from), tables, validate.RequireAll)

	for _, finding := range got.Findings {
		t.Errorf("%s", finding)
	}
	if got.Partial() {
		t.Errorf("skipped %v", got.Skipped)
	}
}

func TestTheBandProjectsShouldDifferFromTheBaseInOneItemOnly(t *testing.T) {
	const varies = "住宅維持費"

	ratios := func(t *testing.T, path string) map[string]string {
		t.Helper()

		table, err := tsv.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("tsv.ReadFile(%s): %v", path, err)
		}
		item, ok := table.ColumnIndex(input.PricedItemColumn)
		if !ok {
			t.Fatalf("%s: no %q column", path, input.PricedItemColumn)
		}
		ratio, ok := table.ColumnIndex(input.InflationRatioColumn)
		if !ok {
			t.Fatalf("%s: no %q column", path, input.InflationRatioColumn)
		}

		got := make(map[string]string, len(table.Rows))
		for _, row := range table.Rows {
			got[row[item]] = row[ratio]
		}
		return got
	}

	want := ratios(t, "data/environment/inflation-target.tsv")

	const (
		lowPath  = "data/environment/scenario/repair-low.tsv"
		highPath = "data/environment/scenario/repair-high.tsv"
	)

	percent := func(t *testing.T, s string) float64 {
		t.Helper()

		move, err := money.ParsePriceMove(s)
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		return move.Rate().Float64()
	}
	low := percent(t, ratios(t, lowPath)[varies])
	high := percent(t, ratios(t, highPath)[varies])
	if adopted := percent(t, want[varies]); !(low < adopted && adopted < high) {
		t.Errorf("幅になっていない: 楽観端 %v・採用値 %v・悲観端 %v の順に大きくなっていない",
			low, adopted, high)
	}

	for name, path := range map[string]string{
		"楽観端": lowPath,
		"悲観端": highPath,
	} {
		t.Run(name, func(t *testing.T) {
			got := ratios(t, path)

			if len(got) != len(want) {
				t.Fatalf("費目が %d 行、%d 行のはず", len(got), len(want))
			}
			for item, ratio := range want {
				switch {
				case item == varies:
					if got[item] == ratio {
						t.Errorf("%s が %s のままである。幅の端が base と同じでは幅にならない", item, ratio)
					}
				case got[item] != ratio:
					t.Errorf("%s が %s である（base は %s）。振るのは %s だけである",
						item, got[item], ratio, varies)
				}
			}
		})
	}
}

func TestTheNumberOfProjectsShouldBeWhatTheRecordsSay(t *testing.T) {
	const counted = 21

	entries, err := os.ReadDir(filepath.Join(repoRoot, "projects"))
	if err != nil {
		t.Fatalf("os.ReadDir: %v", err)
	}
	got := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".tsv" {
			got++
		}
	}

	if got != counted {
		t.Errorf("projects/ に %d 個ある。この検査は %d 個だと言っている。"+
			"**プロジェクトを足したり消したりしたら、それに触れている記録を読み直して測り直すこと**",
			got, counted)
	}
}
