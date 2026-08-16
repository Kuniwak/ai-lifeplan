package ci

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const testWorkflow = "test.yml"

const workflowsDir = "../.github/workflows"

var bucketKey = regexp.MustCompile(`(?m)^ {12}(\w+):$`)

func read(t *testing.T, path string) string {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}

func withoutComments(body string) []string {
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func TestEveryBucketOfChangedFilesIsActedOn(t *testing.T) {
	workflow := read(t, filepath.Join(workflowsDir, testWorkflow))
	keys := bucketKey.FindAllStringSubmatch(workflow, -1)

	if len(keys) < 4 {
		t.Fatalf("want the buckets of changed-files to be found, got %d", len(keys))
	}
	for _, key := range keys {
		output := key[1] + "_any_changed"
		if strings.Count(workflow, output) < 2 {
			t.Errorf("the bucket %q is watched but nothing acts on it: want %s declared as an output and read by at least one job", key[1], output)
		}
	}
}

func TestTheWorkflowWatchesEverythingTheResultsRestOn(t *testing.T) {
	workflow := read(t, filepath.Join(workflowsDir, testWorkflow))

	for _, pattern := range []string{
		"data/**",
		"projects/**",
		"actuals/**",
		"testdata/**",
		"Makefile",
		"pipeline/**",
	} {
		if !strings.Contains(workflow, pattern) {
			t.Errorf("want %q among the paths the workflow watches, or a change to it reports success with nothing checked", pattern)
		}
	}
}

func TestNothingRunInCIAllowsAMissingTable(t *testing.T) {
	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("reading the workflows: %v", err)
	}
	paths := []string{"../Makefile"}
	for _, entry := range entries {
		paths = append(paths, filepath.Join(workflowsDir, entry.Name()))
	}
	if len(paths) < 2 {
		t.Fatalf("want the workflows to be found, got only %v", paths)
	}

	for _, path := range paths {
		for i, line := range withoutComments(read(t, path)) {
			if strings.Contains(line, "-allow-missing") {
				t.Errorf("%s:%d runs -allow-missing: a check that could not run must not be reported as one that passed", filepath.Base(path), i+1)
			}
		}
	}
}

var flaglessValidate = regexp.MustCompile(`\$\([A-Z_]+\) validate [^-\s]`)

func TestTheMakefileRunsTheCheckWithNoFlags(t *testing.T) {
	makefile := read(t, "../Makefile")

	var found []string
	for _, line := range withoutComments(makefile) {
		if flaglessValidate.MatchString(line) {
			found = append(found, strings.TrimSpace(line))
		}
	}

	if len(found) != 1 {
		t.Errorf("want exactly one line running the check with no flags, got %d: %v", len(found), found)
	}
}
