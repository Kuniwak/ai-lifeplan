package tablescmd_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/tablescmd"
)

const repoRoot = "../../.."

func run(t *testing.T, args ...string) (int, *cli.ProcInoutSpy) {
	t.Helper()

	spy := cli.SpyProcInout()
	return tablescmd.NewCommandFunc()(args, spy.NewProcInout()), spy
}

func TestTablesShouldRefuseToWriteOutsideOut(t *testing.T) {
	for _, dir := range []string{"data", "projects", "actuals", "../elsewhere", "."} {
		status, spy := run(t, "-root", repoRoot, "-out", dir, filepath.Join(repoRoot, "projects", "base.tsv"))
		if status == 0 {
			t.Errorf("writing to %q was accepted", dir)
		}
		if !strings.Contains(spy.Stderr.String(), tools.OutRoot) {
			t.Errorf("the error for %q does not say where tables may go: %s", dir, spy.Stderr)
		}
	}
}
