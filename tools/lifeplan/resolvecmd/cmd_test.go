package resolvecmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/cli"
)

const repoRoot = "../../.."

func writeProjects(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"base.tsv": "slot\tpath\n" +
			"household\tdata/household.tsv\n" +
			"income_wife\tdata/income_wife.tsv\n",
		"wife-fulltime.tsv": "slot\tpath\n" +
			"extends\tbase.tsv\n" +
			"income_wife\tdata/income_wife-fulltime.tsv\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return dir
}

func TestResolveShouldListEverySlotWithItsOrigin(t *testing.T) {
	dir := writeProjects(t)
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{filepath.Join(dir, "wife-fulltime.tsv")}, spy.NewProcInout())

	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}

	out := spy.Stdout.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("want a header and two slots, got %d lines:\n%s", len(lines), out)
	}
	if lines[0] != "slot\tpath\torigin" {
		t.Errorf("header = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "household\tdata/household.tsv\t") || !strings.HasSuffix(lines[1], "base.tsv") {
		t.Errorf("household row = %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "income_wife\tdata/income_wife-fulltime.tsv\t") ||
		!strings.HasSuffix(lines[2], "wife-fulltime.tsv") {
		t.Errorf("income_wife row = %q", lines[2])
	}
}

func TestResolveShouldPrintOnlyThePathForASingleSlot(t *testing.T) {
	dir := writeProjects(t)
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{"-slot", "household", filepath.Join(dir, "base.tsv")}, spy.NewProcInout())

	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	if got := spy.Stdout.String(); got != "data/household.tsv\n" {
		t.Errorf("stdout = %q, want just the path", got)
	}
}

func TestResolveShouldApplyASlotOverride(t *testing.T) {
	dir := writeProjects(t)
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{
		"-slot-override", "household=data/household-probe.tsv",
		"-slot", "household",
		filepath.Join(dir, "base.tsv"),
	}, spy.NewProcInout())

	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	if got := spy.Stdout.String(); got != "data/household-probe.tsv\n" {
		t.Errorf("stdout = %q, want the overriding path", got)
	}
}

func TestResolveShouldReportTheCommandLineAsTheOrigin(t *testing.T) {
	dir := writeProjects(t)
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{
		"-slot-override", "household=data/household-probe.tsv",
		filepath.Join(dir, "base.tsv"),
	}, spy.NewProcInout())

	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	if !strings.Contains(spy.Stdout.String(), "data/household-probe.tsv\tcli") {
		t.Errorf("the origin of an overridden slot is not reported as the command line:\n%s", spy.Stdout)
	}
}

func TestResolveNG(t *testing.T) {
	type testCase struct {
		Args     func(dir string) []string
		Mentions string
	}

	testCases := map[string]testCase{
		"no project given": {
			Args:     func(string) []string { return nil },
			Mentions: "argument",
		},
		"more than one project given": {
			Args: func(dir string) []string {
				return []string{filepath.Join(dir, "base.tsv"), filepath.Join(dir, "wife-fulltime.tsv")}
			},
			Mentions: "argument",
		},
		"the project does not exist": {
			Args:     func(dir string) []string { return []string{filepath.Join(dir, "missing.tsv")} },
			Mentions: "missing.tsv",
		},
		"a slot nobody set": {
			Args: func(dir string) []string {
				return []string{"-slot", "no-such-slot", filepath.Join(dir, "base.tsv")}
			},
			Mentions: "no-such-slot",
		},
		"a malformed override": {
			Args: func(dir string) []string {
				return []string{"-slot-override", "household", filepath.Join(dir, "base.tsv")}
			},
			Mentions: "household",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dir := writeProjects(t)
			spy := cli.SpyProcInout()

			status := NewCommandFunc()(tc.Args(dir), spy.NewProcInout())

			if status == 0 {
				t.Fatalf("status = 0, want non-zero (stdout: %q)", spy.Stdout)
			}
			if spy.Stdout.Len() != 0 {
				t.Errorf("standard output must stay empty on failure, got %q", spy.Stdout)
			}
			if !strings.Contains(spy.Stderr.String(), tc.Mentions) {
				t.Errorf("the message does not mention %q: %q", tc.Mentions, spy.Stderr)
			}
		})
	}
}

func TestResolveShouldListEveryFileTheProjectIsReadFrom(t *testing.T) {
	dir := writeProjects(t)
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{"-inputs", "-root", repoRoot, filepath.Join(dir, "wife-fulltime.tsv")}, spy.NewProcInout())

	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}

	var got []string
	for _, path := range strings.Split(strings.TrimRight(spy.Stdout.String(), "\n"), "\n") {
		if strings.HasPrefix(path, filepath.Join(repoRoot, "data", "law")) ||
			strings.HasPrefix(path, filepath.Join(repoRoot, "actuals")) {
			continue
		}
		got = append(got, path)
	}
	want := []string{
		filepath.Join(dir, "base.tsv"),
		filepath.Join(dir, "wife-fulltime.tsv"),
		filepath.Join(repoRoot, "data", "household.tsv"),
		filepath.Join(repoRoot, "data", "income_wife-fulltime.tsv"),
	}
	slices.Sort(want)
	slices.Sort(got)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("inputs mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveWithInputsShouldNameAManifestThatDecidesNothing(t *testing.T) {
	dir := writeProjects(t)
	body := "slot\tpath\nextends\twife-fulltime.tsv\n" +
		"household\tdata/household-own.tsv\n" +
		"income_wife\tdata/income_wife-own.tsv\n"
	if err := os.WriteFile(filepath.Join(dir, "leaf.tsv"), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{"-inputs", "-root", repoRoot, filepath.Join(dir, "leaf.tsv")}, spy.NewProcInout())

	if status != 0 {
		t.Fatalf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	got := spy.Stdout.String()
	for _, want := range []string{
		filepath.Join(dir, "wife-fulltime.tsv"),
		filepath.Join(dir, "base.tsv"),
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("want %s among the files read, got:\n%s", want, got)
		}
	}
}

func TestResolveNGWithBothInputsAndSlot(t *testing.T) {
	dir := writeProjects(t)
	spy := cli.SpyProcInout()

	status := NewCommandFunc()([]string{"-inputs", "-slot", "household", filepath.Join(dir, "base.tsv")}, spy.NewProcInout())

	if status == 0 {
		t.Fatalf("want a refusal, since one asks for a file list and the other for a single path, got:\n%s", spy.Stdout)
	}
	if !strings.Contains(spy.Stderr.String(), "-inputs") || !strings.Contains(spy.Stderr.String(), "-slot") {
		t.Errorf("want the message to name both flags, got: %s", spy.Stderr)
	}
}
