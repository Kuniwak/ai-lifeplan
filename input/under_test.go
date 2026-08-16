package input_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestLoadShouldReadATableGivenByAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "elsewhere.tsv")
	if err := os.WriteFile(path, []byte("西暦\t値\n2030\t1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tables, err := input.Load(".", map[tsv.Slot]string{"probe": path})

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tables["probe"] == nil {
		t.Fatal("the table given by an absolute path was not read")
	}
	if len(tables["probe"].Rows) != 1 {
		t.Errorf("%d row(s), want 1", len(tables["probe"].Rows))
	}
}
