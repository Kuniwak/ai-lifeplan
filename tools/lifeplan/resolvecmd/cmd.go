package resolvecmd

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/version"
)

const (
	SlotColumn   tsv.ColumnName = "slot"
	PathColumn   tsv.ColumnName = "path"
	OriginColumn tsv.ColumnName = "origin"
)

func NewMainFunc() cli.MainFunc[*Options] {
	return func(opts *Options, inout *cli.ProcInout) error {
		if opts.Common.Help {
			return nil
		}
		if opts.Common.Version {
			fmt.Fprintln(inout.Stdout, version.Version)
			return nil
		}

		if opts.Inputs {
			paths, err := plan.InputPaths(plan.Sources{
				Root:          opts.Root,
				ProjectPath:   opts.ProjectPath,
				SlotOverrides: opts.SlotOverrides,
			})
			if err != nil {
				return err
			}
			for _, path := range paths {
				fmt.Fprintln(inout.Stdout, path)
			}
			return nil
		}

		loaded, err := project.Load(opts.ProjectPath)
		if err != nil {
			return err
		}

		layers, err := config.LayersOf(loaded, opts.SlotOverrides)
		if err != nil {
			return err
		}
		settled := config.Resolve(layers)

		if opts.Slot != "" {
			value, ok := settled.Lookup(opts.Slot)
			if !ok {
				return fmt.Errorf("resolvecmd: no layer sets the slot %q in %s", opts.Slot, opts.ProjectPath)
			}
			fmt.Fprintln(inout.Stdout, value.Path)
			return nil
		}

		return tsv.Write(inout.Stdout, tableOf(settled))
	}
}

func tableOf(settled config.Config) *tsv.Table {
	table := &tsv.Table{Header: []tsv.ColumnName{SlotColumn, PathColumn, OriginColumn}}
	for _, slot := range settled.SlotNames() {
		value, _ := settled.Lookup(slot)
		table.Rows = append(table.Rows, []string{string(slot), value.Path, string(value.Origin)})
	}
	return table
}

func NewCommandFunc() cli.CommandFunc {
	return cli.NewCommandFunc(NewParseOptionsFunc(), NewMainFunc())
}
