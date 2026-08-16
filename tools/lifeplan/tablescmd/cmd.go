package tablescmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/version"
)

const Summary = "work out every intermediate table of a project and write them out"

type Options struct {
	Common *tools.CommonOptions

	ProjectPath string

	Root string

	OutDir string

	SlotOverrides map[tsv.Slot]string
}

func NewParseOptionsFunc() cli.ParseOptionsFunc[*Options] {
	return func(args []string, inout *cli.ProcInout) (*Options, error) {
		flags := flag.NewFlagSet("lifeplan tables", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			fmt.Fprintf(flags.Output(), `Usage: lifeplan tables [options] <project.tsv>

%s

One command writes them all. There is no subcommand per table: the dependency
graph belongs to the task runner, and at seventy six
years a whole plan is worked out in milliseconds, so a finer graph would buy
nothing and cost a great deal.

A slot may be replaced from the command line, which beats the manifest
The path may be absolute, so a table can be made on the spot:

  lifeplan tables -slot-override balance=<(qhs -H -O -t -T \
    "select * from actuals/balance.tsv where 西暦 <= 2022") projects/base.tsv

Everything is written under %s/. Nothing writes to data/, projects/ or actuals/.

Options:
`, Summary, tools.OutRoot)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		var overrides tools.SlotOverrideFlag
		tools.DeclareSlotOverride(flags, &overrides)

		root := flags.String("root", ".", "take the paths written in the manifest from here")
		out := flags.String("out", filepath.Join(tools.OutRoot, "base"), "write the tables here")

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &Options{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("tablescmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("tablescmd: %w", err)
		}
		if commonOpts.Version {
			return &Options{Common: tools.CommonOptionsVersion}, nil
		}

		if flags.NArg() != 1 {
			return nil, fmt.Errorf("tablescmd: want exactly one argument, the project to work out, got %d", flags.NArg())
		}
		if err := tools.AssertUnderOut(*out); err != nil {
			return nil, fmt.Errorf("tablescmd: %w", err)
		}

		settled, err := config.ParseSlotOverrides(overrides)
		if err != nil {
			return nil, fmt.Errorf("tablescmd: %w", err)
		}

		return &Options{
			Common:        commonOpts,
			ProjectPath:   flags.Arg(0),
			Root:          *root,
			OutDir:        *out,
			SlotOverrides: settled,
		}, nil
	}
}

func NewMainFunc() cli.MainFunc[*Options] {
	return func(opts *Options, inout *cli.ProcInout) error {
		if opts.Common.Help {
			return nil
		}
		if opts.Common.Version {
			fmt.Fprintln(inout.Stdout, version.Version)
			return nil
		}

		in, err := plan.Load(plan.Sources{
			Root:          opts.Root,
			ProjectPath:   opts.ProjectPath,
			SlotOverrides: opts.SlotOverrides,
		})
		if err != nil {
			return err
		}

		log := tools.NewLogger(opts.Common.LogLevel, inout.Stderr)
		tools.DebugInput(log, opts.Root, opts.ProjectPath)
		tools.WarnUnreadColumns(log, in.UnreadColumns())

		built, err := in.Build()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
			return fmt.Errorf("tablescmd: %w", err)
		}
		if err := built.WriteTables(opts.OutDir); err != nil {
			return err
		}

		fmt.Fprintf(inout.Stderr, "wrote %d tables to %s\n", len(built.Tables()), opts.OutDir)

		if len(built.Uncompared) > 0 {
			years := make([]string, 0, len(built.Uncompared))
			for _, year := range built.Uncompared {
				years = append(years, fmt.Sprint(year))
			}
			fmt.Fprintf(inout.Stderr,
				"収支明細に %s 年のぶんがあるが、残高がその年を持たないので突合していない\n",
				strings.Join(years, "・"))
		}
		return nil
	}
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewParseOptionsFunc(), NewMainFunc())
}
