package validatecmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/tools"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
	"github.com/Kuniwak/lifeplan/version"
)

var ErrFound = fmt.Errorf("the input has findings")

func NewMainFunc() cli.MainFunc[*Options] {
	return func(opts *Options, inout *cli.ProcInout) error {
		if opts.Common.Help {
			return nil
		}
		if opts.Common.Version {
			fmt.Fprintln(inout.Stdout, version.Version)
			return nil
		}

		paths, err := config.SlotPaths(opts.ProjectPath, opts.SlotOverrides)
		if err != nil {
			return err
		}

		tables, err := input.Load(opts.Root, paths)
		if err != nil {
			return err
		}

		log := tools.NewLogger(opts.Common.LogLevel, inout.Stderr)
		tools.DebugInput(log, opts.Root, opts.ProjectPath)
		tools.WarnUnreadColumns(log, input.UnreadColumnsOf(tables))

		planStart, _, err := input.PlanSpan(tables[input.PlanSlot])
		if err != nil {
			return fmt.Errorf("%w (the span of the plan is what every other check is stated over)", err)
		}

		missing := validate.RequireAll
		if opts.AllowMissing {
			missing = validate.AllowMissing
		}

		rules, err := rulesFor(opts, paths, planStart, tables)
		if err != nil {
			return err
		}

		registry := validate.NewRegistry(rules)
		result := validate.Run(rules, tables, missing)

		if err := tsv.Write(inout.Stdout, validate.FindingsTable(result)); err != nil {
			return err
		}
		writeCoverage(inout.Stderr, registry, result)

		if !result.OK() {
			return ErrFound
		}
		return nil
	}
}

func rulesFor(opts *Options, paths map[tsv.Slot]string, planStart date.Year, tables map[tsv.Slot]*tsv.Table) ([]validate.Rule, error) {
	exists := func(string) bool { return true }
	if !opts.AllowMissing {
		exists = func(path string) bool {
			_, err := os.Stat(tsv.Under(opts.Root, path))
			return err == nil
		}
	}

	rules := append(
		input.Rules(planStart),
		validate.SlotResolvable(input.RequiredSlots(), paths, exists),
	)

	fsys := os.DirFS(filepath.Join(opts.Root, filepath.FromSlash(law.LawDirectory)))

	lawTables, err := lawRulesInto(fsys, tables)
	if err != nil {
		return nil, err
	}
	rules = append(rules, lawTables...)

	municipality, err := law.MunicipalityRules(fsys)
	if err != nil {
		return nil, err
	}
	rules = append(rules, municipality...)

	actualsChecks, actualsTables, err := actuals.Rules(opts.Root)
	if err != nil {
		return nil, err
	}
	for slot, table := range actualsTables {
		tables[slot] = table
	}
	return append(rules, actualsChecks...), nil
}

func lawRulesInto(fsys fs.FS, tables map[tsv.Slot]*tsv.Table) ([]validate.Rule, error) {
	rules, lawTables, err := law.Rules(fsys)
	if err != nil {
		return nil, err
	}
	for slot, table := range lawTables {
		tables[slot] = table
	}
	return rules, nil
}

func writeCoverage(w io.Writer, registry validate.Registry, result validate.Result) {
	fmt.Fprintf(w, "checked %d invariant(s), found %d\n", len(result.Ran), len(result.Findings))

	if unexplained := registry.UnwiredWithoutAReason(); len(unexplained) > 0 {
		fmt.Fprintf(w, "\n%d invariant(s) are declared and nothing wires them to a table, with no reason given:\n", len(unexplained))
		for _, d := range unexplained {
			fmt.Fprintf(w, "  %s\n", d.Name)
		}
	}
	if onPurpose := registry.UnwiredOnPurpose(); len(onPurpose) > 0 {
		fmt.Fprintf(w, "\n%d invariant(s) are left unwired on purpose:\n", len(onPurpose))
		for _, d := range onPurpose {
			fmt.Fprintf(w, "  %s — %s\n", d.Name, d.Unwired)
		}
	}

	if stale := registry.WiredWithAReason(); len(stale) > 0 {
		fmt.Fprintf(w, "\n%d invariant(s) say why they are unwired and are wired all the same:\n", len(stale))
		for _, d := range stale {
			fmt.Fprintf(w, "  %s — %s\n", d.Name, d.Unwired)
		}
		fmt.Fprintf(w, "The reason is stale. Remove it from validate.Declarations().\n")
	}

	if !result.Partial() {
		return
	}
	fmt.Fprintf(w, "\ncould not check %d invariant(s), because the tables they read are not there:\n", len(result.Skipped))
	for _, name := range result.Skipped {
		fmt.Fprintf(w, "  %s\n", name)
	}
	fmt.Fprintf(w, "\nThis run is not a complete check. Do not read it as one.\n")
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewParseOptionsFunc(), NewMainFunc())
}
