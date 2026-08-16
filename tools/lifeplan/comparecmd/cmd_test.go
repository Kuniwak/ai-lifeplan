package comparecmd_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/comparecmd"
)

const repoRoot = "../../.."

func run(t *testing.T, args ...string) (int, *cli.ProcInoutSpy) {
	t.Helper()

	spy := cli.SpyProcInout()
	return comparecmd.NewCommandFunc()(args, spy.NewProcInout()), spy
}

func assertDirHolds(t *testing.T, dir string, want []string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	got := make([]string, len(entries))
	for i, entry := range entries {
		got[i] = entry.Name()
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s holds %v, want %v", dir, got, want)
	}
}

func projects(names ...string) []string {
	paths := make([]string, len(names))
	for i, name := range names {
		paths[i] = filepath.Join(repoRoot, "projects", name+".tsv")
	}
	return paths
}

func TestCompareShouldRefuseFewerThanTwoProjects(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out", "compare")
	args := append([]string{"-root", repoRoot, "-out", out}, projects("base")...)

	status, _ := run(t, args...)

	if status == 0 {
		t.Error("status = 0, want non-zero for a single project")
	}
}

func TestCompareShouldRefuseToWriteOutsideOut(t *testing.T) {
	for _, dir := range []string{"data", "projects", "actuals", "../elsewhere", "."} {
		t.Run(dir, func(t *testing.T) {
			args := append([]string{"-root", repoRoot, "-out", dir},
				projects("base", "settle-2050")...)

			status, spy := run(t, args...)

			if status == 0 {
				t.Errorf("writing to %q was accepted", dir)
			}
			if !strings.Contains(spy.Stderr.String(), tools.OutRoot) {
				t.Errorf("the error does not say where tables may go: %s", spy.Stderr)
			}
		})
	}
}

func TestCompareShouldRefuseAnOverrideOfASlotNothingSets(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out", "compare")
	args := append([]string{"-root", repoRoot, "-out", out, "-slot-override", "inflatoin=x.tsv"},
		projects("base", "settle-2050")...)

	status, spy := run(t, args...)

	if status == 0 {
		t.Error("打ち間違えた slot 名の上書きが通った")
	}
	if !strings.Contains(spy.Stderr.String(), "inflatoin") {
		t.Errorf("誤りが名指しされていない: %s", spy.Stderr)
	}
}
