package pipeline

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var skipped = map[string]bool{".git": true, ".wt": true, "out": true, "bin": true}

type sandbox struct {
	root string
}

func newSandbox(t *testing.T) sandbox {
	t.Helper()

	if _, err := exec.LookPath("make"); err != nil {
		t.Skipf("make is not installed: %v", err)
	}

	root := t.TempDir()
	repo := ".."
	err := filepath.WalkDir(repo, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipped[rel] {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		dst := filepath.Join(root, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(dst, content, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copying the repository: %v", err)
	}

	return sandbox{root: root}
}

func (s sandbox) make(t *testing.T, args ...string) string {
	t.Helper()

	cmd := exec.Command("make", args...)
	cmd.Dir = s.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (s sandbox) makeFails(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command("make", args...)
	cmd.Dir = s.root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (s sandbox) path(parts ...string) string {
	return filepath.Join(append([]string{s.root}, parts...)...)
}

func (s sandbox) modTime(t *testing.T, parts ...string) time.Time {
	t.Helper()

	info, err := os.Stat(s.path(parts...))
	if err != nil {
		t.Fatalf("want %s to have been written: %v", filepath.Join(parts...), err)
	}
	return info.ModTime()
}

func (s sandbox) touch(t *testing.T, parts ...string) {
	t.Helper()

	path := s.path(parts...)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("want %s to be an input of some project: %v", filepath.Join(parts...), err)
	}
	now := time.Now()
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("dating %s: %v", filepath.Join(parts...), err)
	}
}

func TestMakeWritesEveryProjectAndTheComparison(t *testing.T) {
	s := newSandbox(t)

	manifests, err := filepath.Glob(filepath.Join("..", "projects", "*.tsv"))
	if err != nil {
		t.Fatalf("projects/ を数える: %v", err)
	}
	if len(manifests) == 0 {
		t.Fatal("projects/ が空である。この検査は何も見ていない")
	}

	want := make([]string, 0, len(manifests)+2)
	for _, manifest := range manifests {
		name := strings.TrimSuffix(filepath.Base(manifest), ".tsv")
		want = append(want, filepath.Join("out", name, "assets.tsv"))
	}
	want = append(want,
		filepath.Join("out", "compare", "summary.tsv"),
		filepath.Join("out", "compare", "assets.svg"))

	s.make(t)

	for _, rel := range want {
		if _, err := os.Stat(s.path(rel)); err != nil {
			t.Errorf("want %s to have been written: %v", rel, err)
		}
	}
}

const shadowing = "shadows-zero-growth"

const shadowingManifest = "slot\tpath\n" +
	"extends\tcase-zero-growth.tsv\n" +
	"inflation\tdata/environment/scenario/inflation-high-growth.tsv\n" +
	"investment_return\tdata/environment/scenario/return-high-growth.tsv\n" +
	"real_wage_growth\tdata/environment/scenario/wage-high-growth.tsv\n" +
	"pension_level\tdata/environment/scenario/pension-high-growth.tsv\n"

func TestMakeRebuildsWhatRestsOnTheChangedInputAndNothingElse(t *testing.T) {
	s := newSandbox(t)
	if err := os.WriteFile(s.path("projects", shadowing+".tsv"), []byte(shadowingManifest), 0o644); err != nil {
		t.Fatalf("writing the project that shadows its parent: %v", err)
	}
	s.make(t)

	for _, tt := range []struct {
		name    string
		touched []string
		rebuilt []string
		left    []string
	}{
		{
			name:    "nothing was changed",
			touched: nil,
			left:    []string{"base", "case-high-growth", "case-zero-growth", shadowing},
		},
		{
			name:    "a scenario table one project reads",
			touched: []string{"data", "environment", "scenario", "inflation-high-growth.tsv"},
			rebuilt: []string{"case-high-growth", shadowing},
			left:    []string{"base", "case-zero-growth"},
		},
		{
			name:    "a manifest in the chain that decides nothing",
			touched: []string{"projects", "case-zero-growth.tsv"},
			rebuilt: []string{"case-zero-growth", "case-zero-growth-depleted", shadowing},
			left:    []string{"base", "case-high-growth"},
		},
		{
			name:    "a law table, which no manifest names",
			touched: []string{"data", "law", "national", "employment-insurance-rate.tsv"},
			rebuilt: []string{"base", "case-high-growth", "case-zero-growth", shadowing},
		},
		{
			name:    "the actuals, which no manifest names either",
			touched: []string{"actuals", "cashflow.tsv"},
			rebuilt: []string{"base", "case-high-growth", "case-zero-growth", shadowing},
		},
		{
			name:    "a record only the checks read",
			touched: []string{"actuals", "securities", "holdings.tsv"},
			rebuilt: []string{"base", "case-high-growth", "case-zero-growth", shadowing},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s.make(t)

			before := make(map[string]time.Time)
			for _, name := range append(append([]string{}, tt.rebuilt...), tt.left...) {
				before[name] = s.modTime(t, "out", name, "assets.tsv")
			}

			if tt.touched != nil {
				s.touch(t, tt.touched...)
			}
			s.make(t)

			for _, name := range tt.rebuilt {
				if after := s.modTime(t, "out", name, "assets.tsv"); !after.After(before[name]) {
					t.Errorf("want the tables of %s to have been worked out again, since it rests on what changed, but they still date from %v", name, after)
				}
			}
			for _, name := range tt.left {
				if after := s.modTime(t, "out", name, "assets.tsv"); !after.Equal(before[name]) {
					t.Errorf("want the tables of %s to have been left alone, since it does not rest on what changed, but they were rewritten at %v (was %v)", name, after, before[name])
				}
			}
		})
	}
}

func TestMakeLeavesNoTableOfAnEarlierRunBehind(t *testing.T) {
	s := newSandbox(t)
	s.make(t)

	ghost := s.path("out", "base", "were-it-still-written.tsv")
	if err := os.WriteFile(ghost, []byte("西暦\t額\n2030\t1\n"), 0o644); err != nil {
		t.Fatalf("writing the table of the earlier run: %v", err)
	}
	s.touch(t, "projects", "base.tsv")
	s.make(t)

	if _, err := os.Stat(ghost); !os.IsNotExist(err) {
		t.Errorf("want the table of the earlier run to be gone, so that it cannot be read as a result of this one: %v", err)
	}
	if _, err := os.Stat(s.path("out", "base", "assets.tsv")); err != nil {
		t.Errorf("want the tables of this run to be there: %v", err)
	}
}

func TestMakeWithAnInputThatDoesNotCheckOutWritesNothing(t *testing.T) {
	s := newSandbox(t)

	broken := filepath.Join("data", "controllable", "allowance-husband.tsv")
	if err := os.WriteFile(s.path(broken), []byte("西暦\t額\nいつか\tいくらか\n"), 0o644); err != nil {
		t.Fatalf("breaking %s: %v", broken, err)
	}

	out, err := s.makeFails(t)
	if err == nil {
		t.Fatalf("want make to have stopped at the check of base, but it succeeded:\n%s", out)
	}
	if want := "validate projects/base.tsv"; !strings.Contains(out, want) {
		t.Errorf("want the check to have run before anything was worked out, and %q to be in what make wrote:\n%s", want, out)
	}
	for _, rel := range []string{
		filepath.Join("out", "base", ".tables"),
		filepath.Join("out", "compare", "summary.tsv"),
	} {
		if _, err := os.Stat(s.path(rel)); !os.IsNotExist(err) {
			t.Errorf("want no %s, since the input it would rest on did not check out: %v", rel, err)
		}
	}
}

func TestMakeAfterTheCodeChangedWorksTheTablesOutAgain(t *testing.T) {
	s := newSandbox(t)
	s.make(t)
	before := s.modTime(t, "out", "base", "assets.tsv")

	source := s.path("money", "money.go")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading %s: %v", source, err)
	}
	const change = "\n// Probe is here to change what the binary does.\nconst Probe = 12345\n"
	if err := os.WriteFile(source, append(body, change...), 0o644); err != nil {
		t.Fatalf("writing %s: %v", source, err)
	}
	s.make(t)

	if after := s.modTime(t, "out", "base", "assets.tsv"); !after.After(before) {
		t.Errorf("want the tables to have been worked out again by the changed code, but they still date from %v", after)
	}
}

func TestMakeNoticesAnInputChangedInTheSameSecondAsTheResults(t *testing.T) {
	s := newSandbox(t)
	s.make(t)
	ranAt := s.modTime(t, "out", "base", "assets.tsv")

	input := s.path("data", "controllable", "allowance-husband.tsv")
	if err := os.Chtimes(input, ranAt, ranAt); err != nil {
		t.Fatalf("dating the input: %v", err)
	}
	s.make(t)

	if after := s.modTime(t, "out", "base", "assets.tsv"); !after.After(ranAt) {
		t.Errorf("want the tables to have been worked out again: the input was written at %v, as the build was writing its tables, and they still date from %v", ranAt, after)
	}
}
