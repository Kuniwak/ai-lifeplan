package lifeplancmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
)

func TestDiffTreesShouldSaySoWhenTheTreesAreTheSame(t *testing.T) {
	tree := map[string][]byte{"a.tsv": []byte("x\n")}

	if got := diffTrees(tree, tree); len(got) != 0 {
		t.Errorf("diffTrees of a tree with itself = %v, want nothing", got)
	}
}

func TestDiffTreesShouldNameAFileWithDifferingBytes(t *testing.T) {
	first := map[string][]byte{"a.tsv": []byte("keep\n1\n")}
	second := map[string][]byte{"a.tsv": []byte("keep\n2\n")}

	got := diffTrees(first, second)

	want := []string{`a.tsv: differs at byte 5 (line 2): "1" vs "2"`}
	if !slices.Equal(got, want) {
		t.Errorf("diffTrees = %v, want %v", got, want)
	}
}

func TestDiffTreesShouldNameAFileOnlyOneRunWrote(t *testing.T) {
	first := map[string][]byte{"only-in-1.tsv": nil}
	second := map[string][]byte{"only-in-2.tsv": nil}

	got := diffTrees(first, second)

	want := []string{
		"only-in-1.tsv: written by the first run only",
		"only-in-2.tsv: written by the second run only",
	}
	if !slices.Equal(got, want) {
		t.Errorf("diffTrees = %v, want %v", got, want)
	}
}

func TestDiffTreesShouldSortWhatItReturns(t *testing.T) {
	first := map[string][]byte{"b.tsv": nil, "a.tsv": nil, "c.tsv": nil}

	got := diffTrees(first, map[string][]byte{})

	if !slices.IsSorted(got) {
		t.Errorf("diffTrees = %v, want it sorted", got)
	}
}

func TestFirstDifferenceShouldTellApartAFileThatMerelyEndsSooner(t *testing.T) {
	got := firstDifference([]byte("x"), []byte("x\n"))

	if got == firstDifference([]byte("x\n"), []byte("x")) {
		t.Errorf("a file ending sooner reads the same either way round: %q", got)
	}
}

func pipelineOutput(t *testing.T, root, workDir, out string) map[string][]byte {
	t.Helper()
	t.Chdir(workDir)

	env := map[string]string{"TZ": "Pacific/Kiritimati", "LANG": "tr_TR.UTF-8", "HOME": "/nonexistent"}
	for name, value := range env {
		t.Setenv(name, value)
	}

	if err := os.RemoveAll(out); err != nil {
		t.Fatalf("cannot clear what the last run wrote: %v", err)
	}

	projects := projectManifests(t, root)
	tree := make(map[string][]byte)
	for _, name := range subcommandsUnderTest(t) {
		for _, inv := range invocations[name](root, out, projects) {
			record(t, tree, env, inv)
		}
	}

	readTree(t, tree, out)
	return tree
}

type invocation struct {
	key  string
	args []string
}

type invocationsFunc func(root, out string, projects []string) []invocation

var invocations = map[string]invocationsFunc{
	"resolve":   perProject("resolve"),
	"validate":  perProject("validate"),
	"tables":    tablesInvocations,
	"breakeven": firstProjectOnly("breakeven"),
	"compare":   compareInvocations,
}

var notRun = map[string]string{
	"import-mf-balance":  "reads a Money Forward export on standard input, which is not in the repository",
	"import-mf-cashflow": "reads a Money Forward export on standard input, which is not in the repository",

	"wizard": "asks questions on standard input and writes a project, so it has nothing to say twice",
}

func perProject(name string) invocationsFunc {
	return func(root, out string, projects []string) []invocation {
		var all []invocation
		for _, project := range projects {
			all = append(all, newInvocation(name, projectName(project), "-root", root, project))
		}
		return all
	}
}

func firstProjectOnly(name string) invocationsFunc {
	return func(root, out string, projects []string) []invocation {
		project := projects[0]
		return []invocation{newInvocation(name, projectName(project), "-root", root, project)}
	}
}

func tablesInvocations(root, out string, projects []string) []invocation {
	var all []invocation
	for _, project := range projects {
		name := projectName(project)
		all = append(all, newInvocation("tables", name, "-root", root, "-out", filepath.Join(out, name), project))
	}
	return all
}

func compareInvocations(root, out string, projects []string) []invocation {
	args := append([]string{"-root", root, "-out", filepath.Join(out, "compare")}, projects...)
	return []invocation{newInvocation("compare", "every project", args...)}
}

func newInvocation(subcommand, over string, args ...string) invocation {
	return invocation{key: subcommand + " " + over, args: append([]string{subcommand}, args...)}
}

func TestEverySubcommandShouldBeAccountedFor(t *testing.T) {
	for _, sub := range Subcommands() {
		_, run := invocations[sub.Name]
		reason, skipped := notRun[sub.Name]
		switch {
		case run && skipped:
			t.Errorf("subcommand %q is both run and listed as not run (%s)", sub.Name, reason)
		case !run && !skipped:
			t.Errorf("subcommand %q is not run by the determinism test and no reason is written down for it", sub.Name)
		}
	}
}

func subcommandsUnderTest(t *testing.T) []string {
	t.Helper()

	var names []string
	for _, sub := range Subcommands() {
		if _, ok := invocations[sub.Name]; ok {
			names = append(names, sub.Name)
		}
	}
	if len(names) == 0 {
		t.Fatal("no subcommand is run, so this test would compare nothing")
	}
	return names
}

func record(t *testing.T, tree map[string][]byte, env map[string]string, inv invocation) {
	t.Helper()

	spy := cli.SpyProcInout()
	spy.Env = env
	status := NewCommandFunc()(inv.args, spy.NewProcInout())

	if _, taken := tree[inv.key+": status"]; taken {
		t.Fatalf("two runs are filed under %q, so one would hide the other", inv.key)
	}
	tree[inv.key+": status"] = []byte(fmt.Sprintf("%d\n", status))
	tree[inv.key+": stdout"] = spy.Stdout.Bytes()
	tree[inv.key+": stderr"] = spy.Stderr.Bytes()
}

func readTree(t *testing.T, tree map[string][]byte, dir string) {
	t.Helper()

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		tree["out/"+filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		t.Fatalf("cannot read what the run wrote: %v", err)
	}
}

func projectManifests(t *testing.T, root string) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(root, "projects", "*.tsv"))
	if err != nil {
		t.Fatalf("cannot list the projects: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("no project under %s/projects, so this test would check nothing", root)
	}
	slices.Sort(paths)
	return paths
}

func projectName(manifest string) string {
	return filepath.Base(manifest[:len(manifest)-len(filepath.Ext(manifest))])
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot tell where this test file is, so the repository cannot be found")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("%s does not look like the repository root: %v", root, err)
	}
	return root
}
