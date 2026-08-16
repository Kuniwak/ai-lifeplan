package validatecmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/validatecmd"
	"github.com/Kuniwak/lifeplan/validate"
)

const repoRoot = "../../.."

func copyOfTheRepositoryInput(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{"data", "projects", "actuals"} {
		if err := os.CopyFS(filepath.Join(root, dir), os.DirFS(filepath.Join(repoRoot, dir))); err != nil {
			t.Fatalf("CopyFS: %v", err)
		}
	}
	return root
}

func run(t *testing.T, args ...string) (status int, spy *cli.ProcInoutSpy) {
	t.Helper()

	spy = cli.SpyProcInout()
	return validatecmd.NewCommandFunc()(args, spy.NewProcInout()), spy
}

func TestValidateShouldPassOnTheInputInTheRepository(t *testing.T) {
	status, spy := run(t, "-root", repoRoot, filepath.Join(repoRoot, "projects", "base.tsv"))

	if status != 0 {
		t.Fatalf("status = %d, want 0\nstdout:\n%s\nstderr:\n%s", status, spy.Stdout, spy.Stderr)
	}
}

func TestValidateShouldReportEveryFindingAtOnce(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	write(t, root, "data/controllable/allowance-husband.tsv",
		"西暦\t小遣い[円/月]\n2018\t34,000\nいつか\t3.4万\n")

	status, spy := run(t, "-root", root, filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", spy.Stdout, spy.Stderr)
	}
	out := spy.Stdout.String()
	for _, want := range []string{"いつか", "3.4万"} {
		if !strings.Contains(out, want) {
			t.Errorf("the findings do not mention %q:\n%s", want, out)
		}
	}
}

func TestValidateShouldFailWhenATableIsNotThere(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	if err := os.Remove(filepath.Join(root, "data", "controllable", "tuition.tsv")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	status, spy := run(t, "-root", root, filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstdout:\n%s\nstderr:\n%s", spy.Stdout, spy.Stderr)
	}
	if out := spy.Stdout.String(); !strings.Contains(out, "tuition") {
		t.Errorf("the findings do not mention the missing table:\n%s", out)
	}
}

func TestValidateShouldNameWhatItSkippedWhenMissingTablesAreAllowed(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	if err := os.Remove(filepath.Join(root, "data", "controllable", "tuition.tsv")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	status, spy := run(t, "-root", root, "-allow-missing", filepath.Join(root, "projects", "base.tsv"))

	if status != 0 {
		t.Fatalf("status = %d, want 0\nstdout:\n%s\nstderr:\n%s", status, spy.Stdout, spy.Stderr)
	}

	report := spy.Stderr.String()
	if !strings.Contains(report, "tuition") {
		t.Errorf("the report does not name the check it skipped:\n%s", report)
	}
}

func TestValidateShouldRefuseWithoutTheSpanOfThePlan(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	if err := os.Remove(filepath.Join(root, "data", "controllable", "plan.tsv")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	status, spy := run(t, "-root", root, "-allow-missing", filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstderr:\n%s", spy.Stderr)
	}
}

func write(t *testing.T, root, path, content string) {
	t.Helper()

	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestValidateShouldCheckTheStatutoryTablesToo(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	spoil(t, root, "data/law/national/standard-remuneration-pension.tsv", func(line string, n int) string {
		if n == 1 {
			return ""
		}
		return line
	})

	status, spy := run(t, "-root", root, filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstdout:\n%s", spy.Stdout)
	}
	if out := spy.Stdout.String(); !strings.Contains(out, "standard-remuneration-pension") {
		t.Errorf("the findings do not name the table with the gap:\n%s", out)
	}
}

func TestValidateShouldRefuseAStatutoryTableWithNoSource(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	spoil(t, root, "data/law/national/employment-insurance-rate.tsv", func(line string, n int) string {
		if n == 0 {
			return line
		}
		fields := strings.Split(line, "\t")
		fields[len(fields)-1] = ""
		return strings.Join(fields, "\t")
	})

	status, spy := run(t, "-root", root, filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstdout:\n%s", spy.Stdout)
	}
	if out := spy.Stdout.String(); !strings.Contains(out, "employment-insurance-rate") {
		t.Errorf("the findings do not name the table with no source:\n%s", out)
	}
}

func TestValidateShouldRefuseAMunicipalityWithNoStatutoryTables(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	write(t, root, "data/environment/residence.tsv", "西暦\t自治体\n2018\t札幌市\n")

	status, spy := run(t, "-root", root, filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstdout:\n%s", spy.Stdout)
	}
	if out := spy.Stdout.String(); !strings.Contains(out, "札幌市") {
		t.Errorf("the findings do not name the municipality with no tables:\n%s", out)
	}
}

func spoil(t *testing.T, root, path string, rewrite func(line string, n int) string) {
	t.Helper()

	full := filepath.Join(root, filepath.FromSlash(path))
	content, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var kept []string
	for n, line := range strings.Split(strings.TrimRight(string(content), "\n"), "\n") {
		if spoiled := rewrite(line, n); spoiled != "" {
			kept = append(kept, spoiled)
		}
	}
	if err := os.WriteFile(full, []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestValidateShouldRefuseAnItemThePlanHasNoPlaceFor(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	spoil(t, root, "actuals/cashflow.tsv", func(line string, n int) string {
		if n != 1 {
			return line
		}
		fields := strings.Split(line, "\t")
		fields[1] = "誰も知らない費目"
		return strings.Join(fields, "\t")
	})

	status, spy := run(t, "-root", root, filepath.Join(root, "projects", "base.tsv"))

	if status == 0 {
		t.Fatalf("status = 0, want non-zero\nstdout:\n%s", spy.Stdout)
	}
	if out := spy.Stdout.String(); !strings.Contains(out, "誰も知らない費目") {
		t.Errorf("the findings do not name it:\n%s", out)
	}
}

func TestValidateShouldReadTheSlotItWasToldToReadInstead(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	broken := filepath.Join("data", "environment", "inflation-broken.tsv")
	if err := os.WriteFile(filepath.Join(root, broken),
		[]byte("西暦\tインフレ率\nいつか\t0.02\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	project := filepath.Join(root, "projects", "base.tsv")

	status, spy := run(t, "-root", root, "-slot-override", "inflation="+filepath.ToSlash(broken), project)

	if status == 0 {
		t.Errorf("差し替えた壊れた表が読まれていない: status = 0\nstdout:\n%s\nstderr:\n%s", spy.Stdout, spy.Stderr)
	}
}

func TestValidateShouldRefuseAnOverrideOfASlotNothingSets(t *testing.T) {
	root := copyOfTheRepositoryInput(t)
	project := filepath.Join(root, "projects", "base.tsv")

	status, spy := run(t, "-root", root, "-slot-override", "inflatoin=x.tsv", project)

	if status == 0 {
		t.Error("打ち間違えた slot 名の上書きが通った")
	}
	if !strings.Contains(spy.Stderr.String(), "inflatoin") {
		t.Errorf("誤りが名指しされていない: %s", spy.Stderr)
	}
}

func TestValidateShouldSayWhichUnwiredRulesHaveAReason(t *testing.T) {
	status, spy := run(t, "-root", repoRoot, filepath.Join(repoRoot, "projects", "base.tsv"))
	if status != 0 {
		t.Fatalf("status = %d, want 0\nstderr:\n%s", status, spy.Stderr)
	}
	out := spy.Stderr.String()

	for _, d := range validate.Declarations() {
		if d.Unwired == "" {
			continue
		}
		if !strings.Contains(out, string(d.Name)) {
			t.Errorf("配線していない %q が出力に無い:\n%s", d.Name, out)
			continue
		}
		if !strings.Contains(out, d.Unwired) {
			t.Errorf("%q の理由が出力に無い:\n%s", d.Name, out)
		}
	}
	if !strings.Contains(out, "left unwired on purpose") {
		t.Errorf("「意図して配線していない」の見出しが無い:\n%s", out)
	}

	if strings.Contains(out, "with no reason given") {
		t.Errorf("理由を書かずに宣言された規則がある。validate.Declarations() を見ること:\n%s", out)
	}
}
