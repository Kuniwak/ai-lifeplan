package breakevencmd_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/breakevencmd"
)

const repoRoot = "../../.."

func run(t *testing.T, args ...string) (int, *cli.ProcInoutSpy) {
	t.Helper()

	spy := cli.SpyProcInout()
	return breakevencmd.NewCommandFunc()(args, spy.NewProcInout()), spy
}

func TestBreakevenShouldTakeASlotOverride(t *testing.T) {
	project := filepath.Join(repoRoot, "projects", "base.tsv")

	status, spy := run(t, "-root", repoRoot,
		"-slot-override", "inflation=data/environment/no-such-table.tsv", project)

	if strings.Contains(spy.Stderr.String(), "flag provided but not defined") {
		t.Fatalf("breakeven が -slot-override を受け付けていない: %s", spy.Stderr)
	}
	if status == 0 {
		t.Errorf("読めない表を差し替えたのに status = 0")
	}
}

func TestBreakevenShouldRefuseAnOverrideOfASlotNothingSets(t *testing.T) {
	project := filepath.Join(repoRoot, "projects", "base.tsv")

	status, spy := run(t, "-root", repoRoot, "-slot-override", "inflatoin=x.tsv", project)

	if status == 0 {
		t.Error("打ち間違えた slot 名の上書きが通った")
	}
	if !strings.Contains(spy.Stderr.String(), "inflatoin") {
		t.Errorf("誤りが名指しされていない: %s", spy.Stderr)
	}
}

func TestBreakevenShouldRefuseDialAndPostponeTogether(t *testing.T) {
	project := filepath.Join(repoRoot, "projects", "base.tsv")

	status, spy := run(t, "-root", repoRoot,
		"-dial", "living_cost:生活費[円/月]", "-postpone", "income_husband",
		"-to", "800", project)

	if status == 0 {
		t.Error("-dial と -postpone が同時に通った")
	}
	if !strings.Contains(spy.Stderr.String(), "-postpone") {
		t.Errorf("誤りが -postpone を名指ししていない: %s", spy.Stderr)
	}
}

func TestBreakevenShouldRefuseNoticeTogetherWithASweepRange(t *testing.T) {
	project := filepath.Join(repoRoot, "projects", "base.tsv")

	status, spy := run(t, "-root", repoRoot,
		"-dial", "living_cost:生活費[円/月]", "-notice", "230000", "-to", "300000", project)

	if status == 0 {
		t.Error("-notice と -to が同時に通った")
	}
	if !strings.Contains(spy.Stderr.String(), "-notice") {
		t.Errorf("誤りが -notice を名指ししていない: %s", spy.Stderr)
	}
}
