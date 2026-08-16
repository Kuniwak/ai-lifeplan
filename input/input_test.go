package input_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestLoadShouldReadTheTableEachSlotPointsAt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "data/controllable/income-husband.tsv", "西暦\t月給収入[円/月]\n2018\t610000\n")

	tables, err := input.Load(root, map[tsv.Slot]string{
		input.IncomeHusbandSlot: "data/controllable/income-husband.tsv",
	})
	if err != nil {
		t.Fatalf("input.Load: %v", err)
	}

	table, ok := tables[input.IncomeHusbandSlot]
	if !ok {
		t.Fatalf("the slot %q was not read", input.IncomeHusbandSlot)
	}
	if got, want := len(table.Rows), 1; got != want {
		t.Errorf("rows = %d, want %d", got, want)
	}
}

func TestLoadShouldLeaveOutASlotWhoseFileIsNotThere(t *testing.T) {
	root := t.TempDir()

	tables, err := input.Load(root, map[tsv.Slot]string{
		input.IncomeHusbandSlot: "data/controllable/income-husband.tsv",
	})
	if err != nil {
		t.Fatalf("input.Load: %v", err)
	}

	if _, ok := tables[input.IncomeHusbandSlot]; ok {
		t.Errorf("the slot %q was read although its file is not there", input.IncomeHusbandSlot)
	}
}

func TestLoadShouldReportEveryFileItCouldNotParse(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.tsv", "西暦\n2018\t610000\n")
	write(t, root, "b.tsv", "西暦\n2018\t610000\n")

	_, err := input.Load(root, map[tsv.Slot]string{"a": "a.tsv", "b": "b.tsv"})
	if err == nil {
		t.Fatal("input.Load returned no error although neither table could be parsed")
	}
	for _, slot := range []string{"a", "b"} {
		if !strings.Contains(err.Error(), slot) {
			t.Errorf("the error does not name the slot %q: %v", slot, err)
		}
	}
}

func write(t *testing.T, root, path, content string) {
	t.Helper()

	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
