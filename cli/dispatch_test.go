package cli

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
)

func echoCommand(got *[]string) CommandFunc {
	return func(args []string, inout *ProcInout) int {
		*got = append(*got, args...)
		return 0
	}
}

func testDispatch(subs ...Subcommand) CommandFunc {
	return NewDispatchFunc("lifeplan", "Life plan simulator.", "1.2.3", subs)
}

func TestDispatchShouldRunTheNamedSubcommand(t *testing.T) {
	var handed []string
	dispatch := testDispatch(
		Subcommand{Name: "validate", Summary: "check the input tables", Run: echoCommand(&handed)},
		Subcommand{Name: "compare", Summary: "compare projects", Run: echoCommand(new([]string))},
	)
	spy := SpyProcInout()

	status := dispatch([]string{"validate", "projects/base.tsv", "-allow-missing"}, spy.NewProcInout())

	if status != 0 {
		t.Errorf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	want := []string{"projects/base.tsv", "-allow-missing"}
	if strings.Join(handed, " ") != strings.Join(want, " ") {
		t.Errorf("the subcommand was handed %v, want %v", handed, want)
	}
}

func TestDispatchShouldReportAnUnknownSubcommand(t *testing.T) {
	dispatch := testDispatch(Subcommand{Name: "validate", Summary: "check", Run: echoCommand(new([]string))})
	spy := SpyProcInout()

	status := dispatch([]string{"valdate"}, spy.NewProcInout())

	if status == 0 {
		t.Error("status = 0, want non-zero for an unknown subcommand")
	}
	if spy.Stdout.Len() != 0 {
		t.Errorf("standard output must carry tables only, got %q", spy.Stdout)
	}
	if !strings.Contains(spy.Stderr.String(), "valdate") {
		t.Errorf("the message does not name the unknown subcommand: %q", spy.Stderr)
	}
}

func TestDispatchWithNoArgumentsShouldFailAndShowTheUsage(t *testing.T) {
	dispatch := testDispatch(Subcommand{Name: "validate", Summary: "check the input tables", Run: echoCommand(new([]string))})
	spy := SpyProcInout()

	status := dispatch(nil, spy.NewProcInout())

	if status == 0 {
		t.Error("status = 0, want non-zero when no subcommand was given")
	}
	if !strings.Contains(spy.Stderr.String(), "validate") {
		t.Errorf("the usage does not list the subcommands: %q", spy.Stderr)
	}
}

func TestDispatchHelpAndVersion(t *testing.T) {
	type testCase struct {
		Args           []string
		WantStdout     string
		StderrMentions []string
	}

	testCases := map[string]testCase{
		"-h lists the subcommands": {
			Args:           []string{"-h"},
			StderrMentions: []string{"lifeplan", "validate", "check the input tables"},
		},
		"--help lists the subcommands": {
			Args:           []string{"--help"},
			StderrMentions: []string{"validate"},
		},
		"-v prints the version": {
			Args:       []string{"-v"},
			WantStdout: "1.2.3\n",
		},
		"--version prints the version": {
			Args:       []string{"--version"},
			WantStdout: "1.2.3\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			dispatch := testDispatch(Subcommand{Name: "validate", Summary: "check the input tables", Run: echoCommand(new([]string))})
			spy := SpyProcInout()

			status := dispatch(tc.Args, spy.NewProcInout())

			if status != 0 {
				t.Errorf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
			}
			if spy.Stdout.String() != tc.WantStdout {
				t.Errorf("stdout = %q, want %q", spy.Stdout, tc.WantStdout)
			}
			for _, want := range tc.StderrMentions {
				if !strings.Contains(spy.Stderr.String(), want) {
					t.Errorf("the usage does not mention %q: %q", want, spy.Stderr)
				}
			}
		})
	}
}

func TestDispatchShouldPassTheStatusOfTheSubcommandThrough(t *testing.T) {
	dispatch := testDispatch(Subcommand{
		Name:    "failing",
		Summary: "always fails",
		Run:     func(args []string, inout *ProcInout) int { return 3 },
	})
	spy := SpyProcInout()

	status := dispatch([]string{"failing"}, spy.NewProcInout())

	if status != 3 {
		t.Errorf("status = %d, want the subcommand's own 3", status)
	}
}

func TestDispatchShouldNotTreatSubcommandFlagsAsItsOwn(t *testing.T) {
	var handed []string
	dispatch := testDispatch(Subcommand{Name: "chart", Summary: "draw", Run: echoCommand(&handed)})
	spy := SpyProcInout()

	status := dispatch([]string{"chart", "-v"}, spy.NewProcInout())

	if status != 0 {
		t.Errorf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	if spy.Stdout.Len() != 0 {
		t.Errorf("the dispatcher answered -v itself: stdout = %q", spy.Stdout)
	}
	if strings.Join(handed, " ") != "-v" {
		t.Errorf("the subcommand was handed %v, want [-v]", handed)
	}
}

func TestNewDispatchFuncShouldPanicOnADuplicatedSubcommandName(t *testing.T) {
	refused := panictest.Recovered(func() {
		testDispatch(
			Subcommand{Name: "validate", Summary: "one", Run: echoCommand(new([]string))},
			Subcommand{Name: "validate", Summary: "two", Run: echoCommand(new([]string))},
		)
	})

	if refused == nil {
		t.Error("want panic for two subcommands sharing a name, got none")
	}
}
