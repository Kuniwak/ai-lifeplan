package cli

import (
	"fmt"
	"io"
)

type Subcommand struct {
	Name string

	Summary string

	Run CommandFunc
}

func NewDispatchFunc(name, summary, version string, subcommands []Subcommand) CommandFunc {
	byName := make(map[string]Subcommand, len(subcommands))
	for _, sub := range subcommands {
		if _, dup := byName[sub.Name]; dup {
			panic(fmt.Sprintf("cli: subcommand %q is registered twice", sub.Name))
		}
		byName[sub.Name] = sub
	}

	return func(args []string, inout *ProcInout) int {
		if len(args) == 0 {
			WriteUsage(inout.Stderr, name, summary, subcommands)
			fmt.Fprintf(inout.Stderr, "\nError: no subcommand given\n")
			return 1
		}

		switch first := args[0]; first {
		case "-h", "--help", "-help":
			WriteUsage(inout.Stderr, name, summary, subcommands)
			return 0

		case "-v", "--version", "-version":
			fmt.Fprintln(inout.Stdout, version)
			return 0

		default:
			sub, ok := byName[first]
			if !ok {
				WriteUsage(inout.Stderr, name, summary, subcommands)
				fmt.Fprintf(inout.Stderr, "\nError: unknown subcommand %q\n", first)
				return 1
			}
			return sub.Run(args[1:], inout)
		}
	}
}

func WriteUsage(w io.Writer, name, summary string, subcommands []Subcommand) {
	fmt.Fprintf(w, "Usage: %s <subcommand> [options]\n\n%s\n\n", name, summary)

	fmt.Fprintf(w, "Subcommands:\n")
	if len(subcommands) == 0 {
		fmt.Fprintf(w, "  (none yet)\n")
	}
	for _, sub := range subcommands {
		fmt.Fprintf(w, "  %-24s %s\n", sub.Name, sub.Summary)
	}

	fmt.Fprintf(w, `
Options:
  -h, --help     show this message
  -v, --version  show version

Run "%s <subcommand> -h" for the options of a subcommand.
`, name)
}
