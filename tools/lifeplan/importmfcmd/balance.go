package importmfcmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/version"
)

const BalanceSummary = "read MoneyForward's 資産推移 export into a table of what it says was held at each year end"

type BalanceOptions struct {
	Common *tools.CommonOptions

	ExportPath, AccountsPath string

	KnownPath string

	OutPath string
}

func NewBalanceParseOptionsFunc() cli.ParseOptionsFunc[*BalanceOptions] {
	return func(args []string, inout *cli.ProcInout) (*BalanceOptions, error) {
		flags := flag.NewFlagSet("lifeplan import-mf-balance", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			fmt.Fprintf(flags.Output(), `Usage: lifeplan import-mf-balance [options] <export.csv>

%s

Only the last day of each year is kept: the export mixes month end rows with
daily ones, and a daily row is not a year's close.

**The export is Shift-JIS as it comes out of MoneyForward** — unlike the
収支明細 one beside it, which is UTF-8 with a byte order mark. Both are read as
they come; a file somebody has already run iconv over reads too. A file that is
neither is refused rather than guessed at.

Every column has to be assigned a bucket in the accounts table. A kind nobody
assigned is refused rather than dropped: dropping it would understate what the
household holds and invent a gap between the plan and the actuals.

Options:
`, BalanceSummary)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		accounts := flags.String("accounts", string(actuals.AccountsPath), "the table saying which bucket each kind of asset goes in")
		known := flags.String("known", string(actuals.KnownPath), "balances recorded elsewhere, for accounts MoneyForward read late or not at all")
		out := flags.String("out", "", "write the table here instead of to standard output")

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &BalanceOptions{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("importmfcmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("importmfcmd: %w", err)
		}
		if commonOpts.Version {
			return &BalanceOptions{Common: tools.CommonOptionsVersion}, nil
		}

		if flags.NArg() != 1 {
			return nil, fmt.Errorf("importmfcmd: want exactly one argument, the export to read, got %d", flags.NArg())
		}

		return &BalanceOptions{
			Common:       commonOpts,
			ExportPath:   flags.Arg(0),
			AccountsPath: *accounts,
			KnownPath:    *known,
			OutPath:      *out,
		}, nil
	}
}

func NewBalanceMainFunc() cli.MainFunc[*BalanceOptions] {
	return func(opts *BalanceOptions, inout *cli.ProcInout) error {
		if opts.Common.Help {
			return nil
		}
		if opts.Common.Version {
			fmt.Fprintln(inout.Stdout, version.Version)
			return nil
		}

		accounts, err := tsv.ReadFile(opts.AccountsPath)
		if err != nil {
			return err
		}
		buckets, err := actuals.ParseBuckets(accounts)
		if err != nil {
			return err
		}

		read, err := tsv.ReadFile(opts.KnownPath)
		if err != nil {
			return err
		}
		known, err := actuals.ParseKnown(read)
		if err != nil {
			return err
		}

		export, err := os.Open(opts.ExportPath)
		if err != nil {
			return fmt.Errorf("importmfcmd: %w", err)
		}
		defer export.Close()

		table, err := actuals.ImportMoneyForwardBalance(export, buckets, known)
		if err != nil {
			return err
		}

		if opts.OutPath == "" {
			return tsv.Write(inout.Stdout, table)
		}
		return tsv.WriteFile(opts.OutPath, table)
	}
}

func NewBalanceCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewBalanceParseOptionsFunc(), NewBalanceMainFunc())
}
