package plan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/tsv"
)

func InputPaths(sources Sources) ([]string, error) {
	loaded, err := project.Load(sources.ProjectPath)
	if err != nil {
		return nil, err
	}

	layers, err := config.LayersOf(loaded, sources.SlotOverrides)
	if err != nil {
		return nil, err
	}

	paths := loaded.Manifests()

	settled := config.Resolve(layers)
	for _, slot := range settled.SlotNames() {
		value, _ := settled.Lookup(slot)
		paths = append(paths, tsv.Under(sources.Root, value.Path))
	}

	unnamed, err := unnamedInputs(sources.Root)
	if err != nil {
		return nil, err
	}
	paths = append(paths, unnamed...)

	slices.Sort(paths)
	return slices.Compact(paths), nil
}

func unnamedInputs(root string) ([]string, error) {
	lawDir := filepath.Join(root, filepath.FromSlash(law.LawDirectory))

	if _, err := os.Stat(lawDir); err != nil {
		return nil, fmt.Errorf("plan.InputPaths: no law tables at %s, and every plan is worked out from them: %w", lawDir, err)
	}

	var paths []string
	err := fs.WalkDir(os.DirFS(lawDir), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".tsv") {
			return nil
		}
		paths = append(paths, filepath.Join(lawDir, filepath.FromSlash(path)))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("plan.InputPaths: reading the law tables: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("plan.InputPaths: %s holds no law table, and every plan is worked out from them", lawDir)
	}

	for _, slot := range []tsv.Slot{
		actuals.CashflowPath, actuals.AdjustmentsPath, actuals.BalancePath,
		actuals.HoldingsPath, actuals.TransactionPath, actuals.OutsidePath,
		actuals.BankBalancePath, actuals.BankAccountsPath, actuals.WifeHoldingsPath, actuals.KnownPath,
		actuals.SourcesPath,
		actuals.AccountsPath,
	} {
		paths = append(paths, tsv.Under(root, string(slot)))
	}
	return paths, nil
}
