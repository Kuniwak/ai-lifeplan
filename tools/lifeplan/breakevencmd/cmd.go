package breakevencmd

import (
	"errors"
	"flag"
	"fmt"

	"github.com/Kuniwak/lifeplan/breakeven"
	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/version"
)

const Summary = "find the rate at which a project stops running out of money"

type Options struct {
	Common *tools.CommonOptions

	ProjectPath string

	Root string

	Dial string

	Postpone string

	From, To, Step int

	SlotOverrides map[tsv.Slot]string

	Notice string
	Until  int
}

func NewParseOptionsFunc() cli.ParseOptionsFunc[*Options] {
	return func(args []string, inout *cli.ProcInout) (*Options, error) {
		flags := flag.NewFlagSet("lifeplan breakeven", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			fmt.Fprintf(flags.Output(), `Usage: lifeplan breakeven [options] <project.tsv>

%s

"How much will inflation be" has no answer. "Above what rate does this
household run out" has one, and it comes from the tables the project already
has.

The whole range is swept rather than searched. A binary search would assume the
answer turns once, and a plan has thresholds in it that can turn it twice.

The dials are turned from the year after the last one the actuals record, and
there is no option to say otherwise. A sweep is about what might happen from
here; it does not rewrite what the household has already earned, and a sweep
that started far enough out would come to the same answer at every setting and
report it as "this parameter does not have to be estimated".

Sweeping a dial replaces whatever the project sets for it, so the inflation
curve of a project that sets 2%% is the same as that of one that sets nought.
What differs between two projects is what the OTHER dials then come to.

With no -dial, the two rates the project carries a range for are swept. Any
other column of a table READ BY YEAR can be swept by naming it:

    lifeplan breakeven -dial living_cost:生活費[円/月] \
        -from 250000 -to 500000 -step 10000 projects/base.tsv

-from, -to and -step are counted in whatever the column is written in: whole
yen for money, hundredths of a per cent for a rate. A column that holds neither
cannot be swept.

Two kinds of table are refused. One read by something other than a year -- a
person, a stage of schooling -- has no "from this year onwards" to give. So has
a list of things that happened: 臨時費用 and 頭金 name a year twice where two
things happened in it, say nothing at all about most years, and a sweep of one
either picked one row of a year where the plan adds them up, or replaced no row
at all and reported that the parameter did not have to be estimated .

A dial named this way has no range worth arguing over written for it, so the
report says where the cliff is and leaves where the household is standing to
the reader.

-postpone is a different kind of correction: not "put this column at X", but
"delay this slot's last row by N years". It names the slot only, with no
column, because 就労延長 has no column to write — turning 給与収入 to a
constant paints over the row that says the household stops working, and turns
a retirement into a household that earns forever . -postpone
leaves every row's value exactly as the project wrote it and moves only the
year the last one takes effect:

    lifeplan breakeven -postpone income_husband -to 10 -step 1 projects/base.tsv

-dial and -postpone do not mix; each names what is being corrected, and naming
both would leave a report answering two different questions.

-notice asks the other question: not how wrong a parameter may be, but **how
late the household may be**. It puts the dial at one setting and runs the YEAR
instead, saying of each year whether making that correction then still removes
the shortfall.

    lifeplan breakeven -dial living_cost:生活費[円/月] \
        -notice 230000 -until 2086 projects/case-zero-growth-depleted.tsv

A 68 year plan does not need 68 years of foresight. Read as a diagnosis rather
than a forecast, being wrong is not the failure — noticing late is .
-until is required because how far ahead is worth asking about is not something
the plan can decide: past the year the household stops working, most corrections
are corrections in name only.

-notice does not mix with -from/-to/-step. Those bound a sweep of the dial, and
this sweeps the year.

-slot-override sweeps against a table the manifest does not name, without
editing the manifest. It is the same flag every other subcommand takes, and it
beats the project. A dial is turned on top of whatever the slot
settled to, so overriding the slot a dial reads changes what is being swept.

Options:
`, Summary)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		root := flags.String("root", ".", "take the paths written in the manifest from here")
		dial := flags.String("dial", "", "sweep this column instead, written slot:列名")
		postpone := flags.String("postpone", "",
			"delay this slot's last row instead of overwriting a column, named by slot only")
		from := flags.Int("from", 0, "sweep from this setting (-dial/-postpone only)")
		to := flags.Int("to", 0, "sweep up to this setting (default 800, in hundredths of a per cent)")
		step := flags.Int("step", breakeven.Step, "sweep in steps of this size")

		notice := flags.String("notice", "", "instead of sweeping, put the dial here and find the last year noticing still helps")
		until := flags.Int("until", 0, "look at noticing up to this year (-notice only)")

		var overrides tools.SlotOverrideFlag
		tools.DeclareSlotOverride(flags, &overrides)

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &Options{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("breakevencmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("breakevencmd: %w", err)
		}
		if commonOpts.Version {
			return &Options{Common: tools.CommonOptionsVersion}, nil
		}

		if flags.NArg() != 1 {
			return nil, fmt.Errorf("breakevencmd: want exactly one argument, the project to sweep, got %d", flags.NArg())
		}
		settled, err := config.ParseSlotOverrides(overrides)
		if err != nil {
			return nil, fmt.Errorf("breakevencmd: %w", err)
		}

		if *dial != "" && *postpone != "" {
			return nil, fmt.Errorf("breakevencmd: -dial と -postpone は一緒に使えない。どちらか一方で何を補正するかを言うこと")
		}
		named := *dial != "" || *postpone != ""

		if *notice != "" {
			if !named {
				return nil, fmt.Errorf("breakevencmd: -notice には -dial か -postpone が要る。どの手を打つのかが決まらない")
			}
			if *from != 0 || *to != 0 || *step != breakeven.Step {
				return nil, fmt.Errorf("breakevencmd: -notice と -from/-to/-step は一緒に使えない。" +
					"-notice はダイヤルを 1 点に置いて年のほうを走らせる")
			}
			if *until == 0 {
				return nil, fmt.Errorf("breakevencmd: -notice には -until が要る。どの年まで見るかは計画が決めることではない")
			}
			return &Options{
				Common:        commonOpts,
				ProjectPath:   flags.Arg(0),
				Root:          *root,
				Dial:          *dial,
				Postpone:      *postpone,
				Notice:        *notice,
				Until:         *until,
				SlotOverrides: settled,
			}, nil
		}

		if !named {
			if *from != 0 || *step != breakeven.Step {
				return nil, fmt.Errorf("breakevencmd: -from and -step only mean something with -dial or -postpone")
			}
			if *to == 0 {
				*to = 800
			}
		} else if *to == 0 {
			return nil, fmt.Errorf("breakevencmd: -to is required with -dial or -postpone, in whatever the dial is written in")
		}
		if *to <= 0 {
			return nil, fmt.Errorf("breakevencmd: -to must be positive, got %d", *to)
		}
		if *step <= 0 {
			return nil, fmt.Errorf("breakevencmd: -step must be positive, got %d", *step)
		}

		return &Options{
			Common:        commonOpts,
			ProjectPath:   flags.Arg(0),
			Root:          *root,
			Dial:          *dial,
			Postpone:      *postpone,
			From:          *from,
			To:            *to,
			Step:          *step,
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

		startsAfter, err := in.StartsAfter()
		if err != nil {
			return err
		}

		if opts.Notice != "" {
			return untilWhen(opts, in, startsAfter+1, inout)
		}

		dials, settings, err := whatToTurn(opts, startsAfter+1)
		if err != nil {
			return err
		}

		swept, err := breakeven.SweepAll(in, dials, settings)
		if err != nil {
			return err
		}
		ranges, err := breakeven.RangesFrom(in)
		if err != nil {
			return err
		}
		for _, line := range breakeven.Report(swept, ranges) {
			fmt.Fprintln(inout.Stderr, line)
		}

		return tsv.Write(inout.Stdout, breakeven.SweptTable(swept))
	}
}

func untilWhen(opts *Options, in *plan.Input, from date.Year, inout *cli.ProcInout) error {
	dial, err := dialFor(opts, from)
	if err != nil {
		return err
	}
	at, err := dial.Kind.Parse(opts.Notice)
	if err != nil {
		return fmt.Errorf("breakevencmd: -notice %q: %w", opts.Notice, err)
	}

	deadlined, err := breakeven.Deadline(in, dial, at, date.Year(opts.Until))
	if err != nil {
		return err
	}

	if warning := breakeven.DeadlineWarning(deadlined); warning != "" {
		fmt.Fprintln(inout.Stderr, warning)
	}
	fmt.Fprintln(inout.Stderr, breakeven.DeadlineSummary(deadlined))
	return tsv.Write(inout.Stdout, breakeven.DeadlineTable(deadlined))
}

func whatToTurn(opts *Options, from date.Year) ([]breakeven.Dial, []breakeven.Setting, error) {
	if opts.Dial == "" && opts.Postpone == "" {
		settings, err := breakeven.Rates(opts.To, opts.Step)
		if err != nil {
			return nil, nil, err
		}
		return breakeven.Dials(from), settings, nil
	}

	dial, err := dialFor(opts, from)
	if err != nil {
		return nil, nil, err
	}
	settings, err := breakeven.Settings(dial.Kind, opts.From, opts.To, opts.Step)
	if err != nil {
		return nil, nil, err
	}
	return []breakeven.Dial{dial}, settings, nil
}

func dialFor(opts *Options, from date.Year) (breakeven.Dial, error) {
	if opts.Postpone != "" {
		return breakeven.PostponeDialOf(tsv.Slot(opts.Postpone), from)
	}
	return breakeven.ParseDial(opts.Dial, from)
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewParseOptionsFunc(), NewMainFunc())
}
