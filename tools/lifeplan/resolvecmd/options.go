package resolvecmd

import (
	"errors"
	"flag"
	"fmt"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
)

const Summary = "show which input table each slot resolves to, and why"

type Options struct {
	Common *tools.CommonOptions

	ProjectPath string

	Slot tsv.Slot

	Inputs bool

	Root string

	SlotOverrides map[tsv.Slot]string
}

func NewParseOptionsFunc() cli.ParseOptionsFunc[*Options] {
	return func(args []string, inout *cli.ProcInout) (*Options, error) {
		flags := flag.NewFlagSet("lifeplan resolve", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			w := flags.Output()
			fmt.Fprintf(w, `Usage: lifeplan resolve [options] <project.tsv>

%s

With no -slot, writes a table of every slot with the path it resolves to and
the layer that decided it. With -slot, writes that one path and nothing else,
so a build file can ask for a dependency:

  $(shell lifeplan resolve -slot income_husband projects/base.tsv)

With -inputs, writes every file the project is read from, one per line: the
table each slot resolves to, every manifest in the chain of extends, the law
tables, and the household's own records. That is more than the table above
shows — a manifest whose slots are all overridden decides nothing and is still
read, and nothing holds a law table in a slot at all — and it is what a build
file needs to know when a result may have gone stale.

Since the law tables and the records are found rather than named, -inputs reads
them from -root, and fails rather than answering short if they are not there.

Options:
`, Summary)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		slot := flags.String("slot", "", "write only this slot's path")
		inputs := flags.Bool("inputs", false, "write every file the project is read from, one per line")
		root := flags.String("root", ".", "take the paths written in the manifest from here")
		var overrides tools.SlotOverrideFlag
		tools.DeclareSlotOverride(flags, &overrides)

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &Options{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("resolvecmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("resolvecmd: %w", err)
		}
		if commonOpts.Version {
			return &Options{Common: tools.CommonOptionsVersion}, nil
		}

		if *inputs && *slot != "" {
			return nil, fmt.Errorf("resolvecmd: -inputs asks for every file the project is read from and -slot for one slot's path, so the two together do not name an answer")
		}

		if flags.NArg() != 1 {
			return nil, fmt.Errorf("resolvecmd: want exactly one argument, the project to resolve, got %d", flags.NArg())
		}

		parsed, err := config.ParseSlotOverrides(overrides)
		if err != nil {
			return nil, fmt.Errorf("resolvecmd: %w", err)
		}

		return &Options{
			Common:        commonOpts,
			ProjectPath:   flags.Arg(0),
			Slot:          tsv.Slot(*slot),
			Inputs:        *inputs,
			Root:          *root,
			SlotOverrides: parsed,
		}, nil
	}
}
