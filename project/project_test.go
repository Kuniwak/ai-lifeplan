package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/tsv"
)

func write(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return dir
}

func TestLoadShouldReadTheSlots(t *testing.T) {
	dir := write(t, map[string]string{
		"base.tsv": "slot\tpath\n" +
			"household\tdata/controllable/household.tsv\n" +
			"market\tdata/environment/market-base.tsv\n",
	})

	got, err := Load(filepath.Join(dir, "base.tsv"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []tsv.Slot{"household", "market"}
	if diff := cmp.Diff(want, got.SlotNames()); diff != "" {
		t.Errorf("SlotNames mismatch (-want +got):\n%s", diff)
	}
	if path, ok := got.Path("household"); !ok || path != "data/controllable/household.tsv" {
		t.Errorf(`Path("household") = (%q, %v)`, path, ok)
	}
}

func TestLoadShouldResolveExtends(t *testing.T) {
	dir := write(t, map[string]string{
		"base.tsv": "slot\tpath\n" +
			"household\tdata/household.tsv\n" +
			"income_wife\tdata/income_wife.tsv\n" +
			"market\tdata/market-base.tsv\n",
		"wife-fulltime.tsv": "slot\tpath\n" +
			"extends\tbase.tsv\n" +
			"income_wife\tdata/income_wife-fulltime.tsv\n",
	})

	got, err := Load(filepath.Join(dir, "wife-fulltime.tsv"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[tsv.Slot]string{
		"household":   "data/household.tsv",
		"income_wife": "data/income_wife-fulltime.tsv",
		"market":      "data/market-base.tsv",
	}
	for slot, wantPath := range want {
		if path, _ := got.Path(slot); path != wantPath {
			t.Errorf("Path(%q) = %q, want %q", slot, path, wantPath)
		}
	}
}

func TestLoadShouldResolveAChainOfExtends(t *testing.T) {
	dir := write(t, map[string]string{
		"base.tsv": "slot\tpath\n" +
			"household\tdata/household.tsv\n" +
			"living_cost\tdata/living_cost.tsv\n",
		"now.tsv": "slot\tpath\n" +
			"extends\tbase.tsv\n" +
			"balance\tactuals/balance.tsv\n",
		"now-slim.tsv": "slot\tpath\n" +
			"extends\tnow.tsv\n" +
			"living_cost\tdata/living_cost-slim.tsv\n",
	})

	got, err := Load(filepath.Join(dir, "now-slim.tsv"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[tsv.Slot]string{
		"household":   "data/household.tsv",
		"balance":     "actuals/balance.tsv",
		"living_cost": "data/living_cost-slim.tsv",
	}
	for slot, wantPath := range want {
		if path, _ := got.Path(slot); path != wantPath {
			t.Errorf("Path(%q) = %q, want %q", slot, path, wantPath)
		}
	}
}

func TestLoadShouldRecordWhereEachSlotWasDecided(t *testing.T) {
	dir := write(t, map[string]string{
		"base.tsv": "slot\tpath\n" +
			"household\tdata/household.tsv\n" +
			"income_wife\tdata/income_wife.tsv\n",
		"wife-fulltime.tsv": "slot\tpath\n" +
			"extends\tbase.tsv\n" +
			"income_wife\tdata/income_wife-fulltime.tsv\n",
	})

	got, err := Load(filepath.Join(dir, "wife-fulltime.tsv"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if src, _ := got.DecidedIn("household"); !strings.HasSuffix(src, "base.tsv") {
		t.Errorf(`DecidedIn("household") = %q, want the base file`, src)
	}
	if src, _ := got.DecidedIn("income_wife"); !strings.HasSuffix(src, "wife-fulltime.tsv") {
		t.Errorf(`DecidedIn("income_wife") = %q, want the overriding file`, src)
	}
}

func TestLoadWithoutExtends(t *testing.T) {
	dir := write(t, map[string]string{
		"standalone.tsv": "slot\tpath\nhousehold\tdata/household.tsv\n",
	})

	got, err := Load(filepath.Join(dir, "standalone.tsv"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if diff := cmp.Diff([]tsv.Slot{"household"}, got.SlotNames()); diff != "" {
		t.Errorf("SlotNames mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadNG(t *testing.T) {
	type testCase struct {
		Files    map[string]string
		Entry    string
		Mentions string
	}

	testCases := map[string]testCase{
		"the file does not exist": {
			Files:    map[string]string{},
			Entry:    "missing.tsv",
			Mentions: "missing.tsv",
		},
		"the parent does not exist": {
			Files:    map[string]string{"child.tsv": "slot\tpath\nextends\tno-such-parent.tsv\na\tb\n"},
			Entry:    "child.tsv",
			Mentions: "no-such-parent.tsv",
		},
		"a project extends itself": {
			Files:    map[string]string{"loop.tsv": "slot\tpath\nextends\tloop.tsv\na\tb\n"},
			Entry:    "loop.tsv",
			Mentions: "loop.tsv",
		},
		"two projects extend each other": {
			Files: map[string]string{
				"a.tsv": "slot\tpath\nextends\tb.tsv\nx\t1\n",
				"b.tsv": "slot\tpath\nextends\ta.tsv\ny\t2\n",
			},
			Entry:    "a.tsv",
			Mentions: "a.tsv",
		},
		"a longer cycle": {
			Files: map[string]string{
				"a.tsv": "slot\tpath\nextends\tb.tsv\nx\t1\n",
				"b.tsv": "slot\tpath\nextends\tc.tsv\ny\t2\n",
				"c.tsv": "slot\tpath\nextends\ta.tsv\nz\t3\n",
			},
			Entry:    "a.tsv",
			Mentions: "a.tsv",
		},
		"the same slot twice in one file": {
			Files:    map[string]string{"dup.tsv": "slot\tpath\nhousehold\tone.tsv\nhousehold\ttwo.tsv\n"},
			Entry:    "dup.tsv",
			Mentions: "household",
		},
		"a slot with no path": {
			Files:    map[string]string{"blank.tsv": "slot\tpath\nhousehold\t\n"},
			Entry:    "blank.tsv",
			Mentions: "household",
		},
		"a path with no slot": {
			Files:    map[string]string{"blank.tsv": "slot\tpath\n\tdata/household.tsv\n"},
			Entry:    "blank.tsv",
			Mentions: "data/household.tsv",
		},
		"more than one extends line": {
			Files:    map[string]string{"two.tsv": "slot\tpath\nextends\ta.tsv\nextends\tb.tsv\nx\t1\n"},
			Entry:    "two.tsv",
			Mentions: "extends",
		},
		"the header is not slot/path": {
			Files:    map[string]string{"odd.tsv": "column1\tcolumn2\nx\t1\n"},
			Entry:    "odd.tsv",
			Mentions: "slot",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dir := write(t, tc.Files)

			_, err := Load(filepath.Join(dir, tc.Entry))

			if err == nil {
				t.Fatal("want error, got none")
			}
			if !strings.Contains(err.Error(), tc.Mentions) {
				t.Errorf("the message does not mention %q: %v", tc.Mentions, err)
			}
		})
	}
}

func TestManifestsShouldNameEveryFileReadEvenWhenNothingItSaysSurvives(t *testing.T) {
	dir := write(t, map[string]string{
		"base.tsv": "slot\tpath\n" +
			"household\tdata/household.tsv\n" +
			"market\tdata/market-base.tsv\n",
		"middle.tsv": "slot\tpath\n" +
			"extends\tbase.tsv\n" +
			"market\tdata/market-middle.tsv\n",
		"leaf.tsv": "slot\tpath\n" +
			"extends\tmiddle.tsv\n" +
			"market\tdata/market-leaf.tsv\n",
	})

	got, err := Load(filepath.Join(dir, "leaf.tsv"))

	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{
		filepath.Join(dir, "leaf.tsv"),
		filepath.Join(dir, "middle.tsv"),
		filepath.Join(dir, "base.tsv"),
	}
	if diff := cmp.Diff(want, got.Manifests()); diff != "" {
		t.Errorf("Manifests mismatch (-want +got):\n%s", diff)
	}
}
