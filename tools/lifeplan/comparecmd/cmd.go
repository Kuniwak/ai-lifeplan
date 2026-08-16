package comparecmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/version"
)

const Summary = "set two or more projects side by side and say what differs and why"

type Options struct {
	Common *tools.CommonOptions

	ProjectPaths []string

	Root string

	OutDir string

	SlotOverrides map[tsv.Slot]string
}

func NewParseOptionsFunc() cli.ParseOptionsFunc[*Options] {
	return func(args []string, inout *cli.ProcInout) (*Options, error) {
		flags := flag.NewFlagSet("lifeplan compare", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			fmt.Fprintf(flags.Output(), `Usage: lifeplan compare [options] <base.tsv> <other.tsv>...

%s

The first project is the one the others are measured against, and its 実績 table
is the record every project is checked against. Six tables are written:

  summary.tsv   one row per project: the year the assets run out, and the
                lifetime totals it holds or fails by
  timeline.tsv  the headline metrics of every project, year by year
  diff.tsv      every field of every intermediate table the projects disagree
                on, against the first. The tables are joined on the year, so
                projects recording different spans are still compared over the
                years they share; 年の有無 says of each row whether both have
                that year or only one of them does
  slots.tsv     the input tables they differ in, and whether the household
                chose them
  actual-vs-plan.tsv
                for every year the plan and the record both reach, what each
                project planned to hold and what was actually held, per bucket.
                It is empty for a project starting from the latest record —
                its plan begins where the record ends, so there is nothing yet
                to check
  spending-vs-plan.tsv
                for every year the record's own inputs answer for, what each
                project's plan expected to spend and what the record says was
                actually spent, worked out from the balances
  assets.svg    the holdings of every project drawn over time, with the year
                each runs out marked by name and the record overlaid as a
                dashed line

-slot-override compares against a table no manifest names, without editing one.
It is the same flag every other subcommand takes, and it beats the project
It is applied to every project, so what it changes is the ground
the projects are compared on rather than one of them; naming a project per
override is not built. Because it lands on every project it leaves no difference
between them, so slots.tsv lists it under the class コマンド引数 and the report
says so — otherwise nothing written down would say which ground the numbers
stand on.

Everything is written under %s/. Nothing writes to data/, projects/ or actuals/.

Options:
`, Summary, tools.OutRoot)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		root := flags.String("root", ".", "take the paths written in the manifests from here")
		out := flags.String("out", filepath.Join(tools.OutRoot, "compare"), "write the comparison here")

		var overrides tools.SlotOverrideFlag
		tools.DeclareSlotOverride(flags, &overrides)

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &Options{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("comparecmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("comparecmd: %w", err)
		}
		if commonOpts.Version {
			return &Options{Common: tools.CommonOptionsVersion}, nil
		}

		if flags.NArg() < 2 {
			return nil, fmt.Errorf(
				"comparecmd: want at least two projects to compare, got %d", flags.NArg())
		}
		if err := tools.AssertUnderOut(*out); err != nil {
			return nil, fmt.Errorf("comparecmd: %w", err)
		}

		settled, err := config.ParseSlotOverrides(overrides)
		if err != nil {
			return nil, fmt.Errorf("comparecmd: %w", err)
		}

		return &Options{
			Common:        commonOpts,
			ProjectPaths:  flags.Args(),
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

		log := tools.NewLogger(opts.Common.LogLevel, inout.Stderr)
		for _, path := range opts.ProjectPaths {
			tools.DebugInput(log, opts.Root, path)
		}

		subjects, err := compare.Load(compare.Sources{
			Root:          opts.Root,
			ProjectPaths:  opts.ProjectPaths,
			SlotOverrides: opts.SlotOverrides,
		})
		if err != nil {
			return err
		}
		for _, subject := range subjects {
			tools.WarnUnreadColumns(log, subject.UnreadColumns)
		}
		compared, err := compare.Of(subjects)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
			return fmt.Errorf("comparecmd: %w", err)
		}
		if err := compared.WriteTables(opts.OutDir); err != nil {
			return err
		}
		if err := compared.WriteChart(opts.OutDir); err != nil {
			return err
		}

		fmt.Fprintf(inout.Stderr, "wrote %d tables and %s to %s\n",
			len(compared.Tables()), compare.ChartFile, opts.OutDir)

		for _, warning := range compared.Warnings {
			fmt.Fprintln(inout.Stderr, warning)
		}
		return nil
	}
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewParseOptionsFunc(), NewMainFunc())
}
