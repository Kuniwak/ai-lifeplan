package lifeplancmd

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/version"
)

func TestNewCommandFuncShouldPrintVersion(t *testing.T) {
	cmdFunc := NewCommandFunc()
	spy := cli.SpyProcInout()

	status := cmdFunc([]string{"-v"}, spy.NewProcInout())

	if status != 0 {
		t.Errorf("status = %d, want 0 (stderr: %s)", status, spy.Stderr)
	}
	if want := version.Version + "\n"; spy.Stdout.String() != want {
		t.Errorf("stdout = %q, want %q", spy.Stdout, want)
	}
}

func TestNewCommandFuncShouldShowTheUsage(t *testing.T) {
	cmdFunc := NewCommandFunc()
	spy := cli.SpyProcInout()

	status := cmdFunc([]string{"-h"}, spy.NewProcInout())

	if status != 0 {
		t.Errorf("status = %d, want 0", status)
	}
	if !strings.Contains(spy.Stderr.String(), "lifeplan") {
		t.Errorf("the usage does not name the tool: %q", spy.Stderr)
	}
	if spy.Stdout.Len() != 0 {
		t.Errorf("standard output must carry tables only, got %q", spy.Stdout)
	}
}

func TestNewCommandFuncShouldRejectAnUnknownSubcommand(t *testing.T) {
	cmdFunc := NewCommandFunc()
	spy := cli.SpyProcInout()

	status := cmdFunc([]string{"no-such-subcommand"}, spy.NewProcInout())

	if status == 0 {
		t.Error("status = 0, want non-zero for an unknown subcommand")
	}
	if !strings.Contains(spy.Stderr.String(), "no-such-subcommand") {
		t.Errorf("the message does not name the unknown subcommand: %q", spy.Stderr)
	}
}

func TestSubcommandsShouldBeUniquelyNamed(t *testing.T) {
	seen := make(map[string]struct{})

	for _, sub := range Subcommands() {
		if sub.Name == "" {
			t.Error("a subcommand has no name")
		}
		if sub.Summary == "" {
			t.Errorf("subcommand %q has no summary, so the usage cannot describe it", sub.Name)
		}
		if sub.Run == nil {
			t.Errorf("subcommand %q has nothing to run", sub.Name)
		}
		if _, dup := seen[sub.Name]; dup {
			t.Errorf("subcommand %q is listed twice", sub.Name)
		}
		seen[sub.Name] = struct{}{}
	}
}
