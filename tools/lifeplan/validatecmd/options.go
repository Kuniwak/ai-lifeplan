package validatecmd

import (
	"errors"
	"flag"
	"fmt"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
)

const Summary = "check the input tables of a project before anything is computed from them"

type Options struct {
	Common *tools.CommonOptions

	ProjectPath string

	Root string

	AllowMissing bool

	SlotOverrides map[tsv.Slot]string
}

func NewParseOptionsFunc() cli.ParseOptionsFunc[*Options] {
	return func(args []string, inout *cli.ProcInout) (*Options, error) {
		flags := flag.NewFlagSet("lifeplan validate", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			w := flags.Output()
			fmt.Fprintf(w, `Usage: lifeplan validate [options] <project.tsv>

%s

Which checks run follows from the tables that are there; there is nothing to
name on the command line. Writes the findings as a table on standard output and
what was checked on standard error. Exits non-zero when anything was found.

Without -allow-missing, a table that is not there is itself a finding: a check
that could not run must never be reported as one that passed. -allow-missing is
for input that is still being written, and always names what it skipped. Do not
use it in CI.

-slot-override checks a table the manifest does not name, without editing the
manifest. It is the same flag every other subcommand takes, and it beats the
project. A slot no layer sets is refused rather than added.

Options:
`, Summary)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		root := flags.String("root", ".", "take the paths written in the manifest from here")
		allowMissing := flags.Bool("allow-missing", false, "skip the checks whose tables are not there, and name them")

		var overrides tools.SlotOverrideFlag
		tools.DeclareSlotOverride(flags, &overrides)

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &Options{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("validatecmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("validatecmd: %w", err)
		}
		if commonOpts.Version {
			return &Options{Common: tools.CommonOptionsVersion}, nil
		}

		if flags.NArg() != 1 {
			return nil, fmt.Errorf("validatecmd: want exactly one argument, the project to check, got %d", flags.NArg())
		}

		settled, err := config.ParseSlotOverrides(overrides)
		if err != nil {
			return nil, fmt.Errorf("validatecmd: %w", err)
		}

		return &Options{
			Common:        commonOpts,
			ProjectPath:   flags.Arg(0),
			Root:          *root,
			AllowMissing:  *allowMissing,
			SlotOverrides: settled,
		}, nil
	}
}
