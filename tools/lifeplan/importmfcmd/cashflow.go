package importmfcmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/version"
)

const CashflowSummary = "read MoneyForward's 収入・支出詳細 export into a table of one month and one item a row"

type CashflowOptions struct {
	Common *tools.CommonOptions

	ExportPaths []string

	CategoriesPath string

	PayeesPath string

	ExcludedPath string

	HeldAccountsPath string

	SourcesPath string

	UntrustedPath string

	OutPath string
}

func NewCashflowParseOptionsFunc() cli.ParseOptionsFunc[*CashflowOptions] {
	return func(args []string, inout *cli.ProcInout) (*CashflowOptions, error) {
		flags := flag.NewFlagSet("lifeplan import-mf-cashflow", flag.ContinueOnError)
		flags.SetOutput(inout.Stderr)
		flags.Usage = func() {
			fmt.Fprintf(flags.Output(), `Usage: lifeplan import-mf-cashflow [options] <export.csv>...

%s

The export is one file a year and several may be given at once; the rows are
summed by month and item whichever file they came from.

計算対象 = 0 is not taken as an answer. MoneyForward lets a row be called a
transfer without the other side of it being anywhere, so the mark says what the
household clicked rather than what the money did; the excluded table decides
what each such 中項目 is, and one that is not in it is refused. Every
中項目 has to be mapped to an item in the categories table: one that is not is
refused rather than dropped, because a spending line dropped is spending the
plan never sees.

Options:
`, CashflowSummary)
			flags.PrintDefaults()
		}

		var commonRawOpts tools.CommonRawOptions
		tools.DeclareCommonOptions(flags, &commonRawOpts)

		categories := flags.String("categories", "actuals/categories.tsv", "the table saying which item each 中項目 belongs to")
		payees := flags.String("payees", string(actuals.PayeesPath), "the table of payees whose item their category cannot decide; its 観測 column is written back")
		excluded := flags.String("excluded", string(actuals.ExcludedPath), "the table saying what to do with the rows MoneyForward has taken out of the calculation; its 観測 column is written back")
		held := flags.String("mf-accounts", string(actuals.HeldAccountsPath), "the table of accounts MoneyForward keeps its own ledger for; money moved into one of them is a transfer")
		sources := flags.String("sources", string(actuals.SourcesPath), "write the record of which exports were read here")
		untrusted := flags.String("untrusted", string(actuals.UntrustedPath), "write what each untrusted account's own ledger says here")
		out := flags.String("out", "", "write the table here instead of to standard output")

		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return &CashflowOptions{Common: tools.CommonOptionsHelp}, nil
			}
			return nil, fmt.Errorf("importmfcmd: %w", err)
		}

		commonOpts, err := tools.ValidateCommonOptions(&commonRawOpts)
		if err != nil {
			return nil, fmt.Errorf("importmfcmd: %w", err)
		}
		if commonOpts.Version {
			return &CashflowOptions{Common: tools.CommonOptionsVersion}, nil
		}

		if flags.NArg() == 0 {
			return nil, fmt.Errorf("importmfcmd: want at least one argument, an export to read")
		}

		return &CashflowOptions{
			Common:           commonOpts,
			ExportPaths:      flags.Args(),
			CategoriesPath:   *categories,
			PayeesPath:       *payees,
			ExcludedPath:     *excluded,
			HeldAccountsPath: *held,
			SourcesPath:      *sources,
			UntrustedPath:    *untrusted,
			OutPath:          *out,
		}, nil
	}
}

func NewCashflowMainFunc() cli.MainFunc[*CashflowOptions] {
	return func(opts *CashflowOptions, inout *cli.ProcInout) error {
		if opts.Common.Help {
			return nil
		}
		if opts.Common.Version {
			fmt.Fprintln(inout.Stdout, version.Version)
			return nil
		}

		read, err := tsv.ReadFile(opts.CategoriesPath)
		if err != nil {
			return err
		}
		categories, err := actuals.ImportRulesFromCategories(read)
		if err != nil {
			return err
		}

		payees, err := tsv.ReadFile(opts.PayeesPath)
		if err != nil {
			return err
		}
		if categories, err = categories.WithPayees(payees); err != nil {
			return err
		}

		excludedTable, err := tsv.ReadFile(opts.ExcludedPath)
		if err != nil {
			return err
		}
		if categories, err = categories.WithExcluded(excludedTable); err != nil {
			return err
		}

		heldTable, err := tsv.ReadFile(opts.HeldAccountsPath)
		if err != nil {
			return err
		}
		if categories, err = categories.WithHeldAccounts(heldTable); err != nil {
			return err
		}

		files := make([]actuals.CashflowFile, 0, len(opts.ExportPaths))
		for _, path := range opts.ExportPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("importmfcmd: %w", err)
			}
			files = append(files, actuals.CashflowFile{Name: filepath.Base(path), Content: content})
		}

		imported, err := actuals.ImportCashflowFiles(files, categories, excludedTable, payees)
		if err != nil {
			return err
		}

		if err := tsv.WriteFile(opts.ExcludedPath, imported.Excluded); err != nil {
			return err
		}
		if err := tsv.WriteFile(opts.PayeesPath, imported.Payees); err != nil {
			return err
		}
		if err := tsv.WriteFile(opts.SourcesPath, actuals.SourceTable(imported.Sources)); err != nil {
			return err
		}

		if err := tsv.WriteFile(opts.UntrustedPath, imported.Untrusted); err != nil {
			return err
		}

		if opts.OutPath == "" {
			return tsv.Write(inout.Stdout, imported.Cashflow)
		}
		return tsv.WriteFile(opts.OutPath, imported.Cashflow)
	}
}

func NewCashflowCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewCashflowParseOptionsFunc(), NewCashflowMainFunc())
}
