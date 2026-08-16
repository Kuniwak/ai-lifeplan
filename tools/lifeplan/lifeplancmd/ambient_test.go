package lifeplancmd

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var ambientPackages = map[string]string{
	"time":          "a result that depends on the clock cannot be reproduced tomorrow; take the year from the input tables",
	"math/rand":     "a projection is worked out, not sampled",
	"math/rand/v2":  "a projection is worked out, not sampled",
	"crypto/rand":   "a projection is worked out, not sampled",
	"net":           "every input is a file in this repository, and a result needing the network could not be checked offline",
	"net/http":      "every input is a file in this repository, and a result needing the network could not be checked offline",
	"os/user":       "a result must not differ between two people running it",
	"runtime/debug": "build information reaches the output and differs between builds of the same source",
}

func TestNothingComputedShouldReachForAnAmbientAnswer(t *testing.T) {
	root := repoRoot(t)
	files := nonTestGoFiles(t, root)
	fset := token.NewFileSet()

	for _, path := range files {
		parsed, err := parser.ParseFile(fset, filepath.Join(root, path), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("cannot read %s: %v", path, err)
		}
		for _, imported := range parsed.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatalf("%s: cannot read an import path: %v", path, err)
			}
			if why, banned := ambientPackages[name]; banned {
				t.Errorf("%s imports %q: %s", path, name, why)
			}
		}
	}
}

func TestTheAmbientCheckShouldReachTheCodeThatComputes(t *testing.T) {
	root := repoRoot(t)

	files := nonTestGoFiles(t, root)

	for _, want := range []string{
		filepath.Join("tools", "lifeplan", "lifeplancmd", "cmd.go"),
		filepath.Join("cli", "cli.go"),
		filepath.Join("tsv", "file.go"),
	} {
		if !slices.Contains(files, want) {
			t.Errorf("the check does not reach %s, which is code a result is worked out by", want)
		}
	}
	for _, unwanted := range files {
		if strings.HasSuffix(unwanted, "_test.go") {
			t.Errorf("%s is a test file and the check must not hold it to the ban", unwanted)
		}
	}
}

func nonTestGoFiles(t *testing.T, root string) []string {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != root && (strings.HasPrefix(name, ".") || name == "out" || name == "testdata") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("cannot read the source: %v", err)
	}
	slices.Sort(paths)
	return paths
}
