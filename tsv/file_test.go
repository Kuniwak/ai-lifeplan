package tsv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func leftovers(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestWriteFileShouldLeaveOnlyTheTargetBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "timeline.tsv")
	table := &Table{
		Header: []ColumnName{"西暦", "収支"},
		Rows:   [][]string{{"2031", "2270000"}},
	}

	if err := WriteFile(path, table); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if diff := cmp.Diff([]string{"timeline.tsv"}, leftovers(t, dir)); diff != "" {
		t.Errorf("the directory holds more than the target (-want +got):\n%s", diff)
	}

	got, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if diff := cmp.Diff(table, got); diff != "" {
		t.Errorf("what was written differs from what was read (-want +got):\n%s", diff)
	}
}

func TestWriteFileShouldNotTouchAnExistingFileWhenTheTableIsInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "timeline.tsv")
	previous := "西暦\t収支\n2031\t2270000\n"
	if err := os.WriteFile(path, []byte(previous), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	broken := &Table{
		Header: []ColumnName{"西暦", "収支"},
		Rows:   [][]string{{"2031"}},
	}

	err := WriteFile(path, broken)

	if err == nil {
		t.Fatal("want error for a row that does not match the header, got none")
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("ReadFile: %v", readErr)
	}
	if string(after) != previous {
		t.Errorf("the previous output was disturbed:\nwant %q\ngot  %q", previous, string(after))
	}
	if diff := cmp.Diff([]string{"timeline.tsv"}, leftovers(t, dir)); diff != "" {
		t.Errorf("a temporary file was left behind (-want +got):\n%s", diff)
	}
}

func TestWriteFileShouldReplaceAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "timeline.tsv")
	if err := os.WriteFile(path, []byte("古い\n内容\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	table := &Table{Header: []ColumnName{"西暦"}, Rows: [][]string{{"2031"}}}

	if err := WriteFile(path, table); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if want := "西暦\n2031\n"; string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestWriteFileShouldProduceTheSameBytesEveryRun(t *testing.T) {
	dir := t.TempDir()
	table := &Table{
		Header: []ColumnName{"西暦", "貯蓄", "金融資産", "資産合計"},
		Rows: [][]string{
			{"2031", "1369000", "14788000", "16156000"},
			{"2032", "1400000", "15000000", "16400000"},
			{"2033", "1450000", "15500000", "16950000"},
		},
	}

	var runs [][]byte
	for i := range 2 {
		path := filepath.Join(dir, "assets"+string(rune('a'+i))+".tsv")
		if err := WriteFile(path, table); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		runs = append(runs, b)
	}

	if diff := cmp.Diff(string(runs[0]), string(runs[1])); diff != "" {
		t.Errorf("two runs produced different bytes (-first +second):\n%s", diff)
	}
}

func TestReadFileNG(t *testing.T) {
	dir := t.TempDir()

	_, err := ReadFile(filepath.Join(dir, "does-not-exist.tsv"))

	if err == nil {
		t.Error("want error for a missing file, got none")
	}
}
