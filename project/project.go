package project

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
)

const ExtendsSlot = "extends"

const (
	SlotColumn tsv.ColumnName = "slot"
	PathColumn tsv.ColumnName = "path"
)

type Project struct {
	path string

	paths map[tsv.Slot]string

	decidedIn map[tsv.Slot]string

	manifests []string
}

func (p *Project) Path(slot tsv.Slot) (string, bool) {
	v, ok := p.paths[slot]
	return v, ok
}

func (p *Project) DecidedIn(slot tsv.Slot) (string, bool) {
	v, ok := p.decidedIn[slot]
	return v, ok
}

func (p *Project) SlotNames() []tsv.Slot {
	return slices.Sorted(maps.Keys(p.paths))
}

func (p *Project) Manifests() []string {
	return slices.Clone(p.manifests)
}

func (p *Project) ManifestPath() string {
	return p.path
}

func Load(path string) (*Project, error) {
	return load(path, nil)
}

func load(path string, chain []string) (*Project, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("project.Load: %s: %w", path, err)
	}

	if i := slices.Index(chain, abs); i >= 0 {
		return nil, fmt.Errorf("project.Load: %s extends itself through %v", path, append(chain[i:], abs))
	}

	table, err := tsv.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("project.Load: %w", err)
	}

	own, parent, err := readRows(path, table)
	if err != nil {
		return nil, err
	}

	resolved := &Project{
		path:      path,
		paths:     make(map[tsv.Slot]string, len(own)),
		decidedIn: make(map[tsv.Slot]string, len(own)),
		manifests: []string{path},
	}

	if parent != "" {
		parentPath := filepath.Join(filepath.Dir(path), parent)
		inherited, err := load(parentPath, append(chain, abs))
		if err != nil {
			return nil, err
		}
		for _, slot := range inherited.SlotNames() {
			resolved.paths[slot], _ = inherited.Path(slot)
			resolved.decidedIn[slot], _ = inherited.DecidedIn(slot)
		}
		resolved.manifests = append(resolved.manifests, inherited.Manifests()...)
	}

	for slot, slotPath := range own {
		resolved.paths[slot] = slotPath
		resolved.decidedIn[slot] = path
	}

	return resolved, nil
}

func readRows(path string, table *tsv.Table) (slots map[tsv.Slot]string, parent string, err error) {
	slotColumn, ok := table.ColumnIndex(SlotColumn)
	if !ok {
		return nil, "", fmt.Errorf("project.Load: %s: no %q column", path, SlotColumn)
	}
	pathColumn, ok := table.ColumnIndex(PathColumn)
	if !ok {
		return nil, "", fmt.Errorf("project.Load: %s: no %q column", path, PathColumn)
	}

	slots = make(map[tsv.Slot]string, len(table.Rows))
	for i, row := range table.Rows {
		slot, slotPath := row[slotColumn], row[pathColumn]

		if slot == "" {
			return nil, "", fmt.Errorf("project.Load: %s: row %d has a path %q but no slot", path, i+1, slotPath)
		}
		if slotPath == "" {
			return nil, "", fmt.Errorf("project.Load: %s: slot %q has no path (leave the row out rather than blank)", path, slot)
		}

		if slot == ExtendsSlot {
			if parent != "" {
				return nil, "", fmt.Errorf("project.Load: %s: %q appears more than once, so which project is extended is undecidable", path, ExtendsSlot)
			}
			parent = slotPath
			continue
		}

		if _, dup := slots[tsv.Slot(slot)]; dup {
			return nil, "", fmt.Errorf("project.Load: %s: slot %q appears more than once", path, slot)
		}
		slots[tsv.Slot(slot)] = slotPath
	}

	return slots, parent, nil
}
