package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Request struct {
	Input *plan.Input

	Base string

	Root string

	Dir, Name string

	Answers []Answer
}

func Write(r Request) (string, error) {
	if r.Name == "" {
		return "", fmt.Errorf("wizard.Write: 名前が空である")
	}
	if r.Base == "" {
		return "", fmt.Errorf("wizard.Write: extends する manifest が空である")
	}
	under := filepath.Join(r.Root, r.Dir)
	if err := os.MkdirAll(under, 0o777); err != nil {
		return "", fmt.Errorf("wizard.Write: %w", err)
	}

	chosen := make(map[tsv.Slot]string, len(r.Answers))

	turned := make(map[tsv.Slot]*tsv.Table, len(r.Answers))
	order := make([]tsv.Slot, 0, len(r.Answers))
	for _, a := range r.Answers {
		if a.Question.AnsweredByTable() {
			if _, seen := chosen[a.Choice.Slot]; !seen {
				order = append(order, a.Choice.Slot)
			}
			chosen[a.Choice.Slot] = a.Choice.Path
			continue
		}
		slot := a.Question.Dial.Slot

		written, ok := turned[slot]
		if !ok {
			var err error
			if written, err = r.Input.Table(slot); err != nil {
				return "", fmt.Errorf("wizard.Write: %s: %w", a.Question.Name, err)
			}
			order = append(order, slot)
		}

		next, err := a.Question.Dial.Turn(written, a.Setting)
		if err != nil {
			return "", fmt.Errorf("wizard.Write: %s: %w", a.Question.Name, err)
		}
		turned[slot] = next
	}

	rows := make([][]string, 0, len(order)+1)
	rows = append(rows, []string{string(project.ExtendsSlot), r.Base})
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	for _, slot := range order {
		if path, ok := chosen[slot]; ok {
			rows = append(rows, []string{string(slot), path})
			continue
		}
		name := fmt.Sprintf("%s-%s.tsv", r.Name, slot)
		if err := tsv.WriteFile(filepath.Join(under, name), turned[slot]); err != nil {
			return "", fmt.Errorf("wizard.Write: %w", err)
		}
		rows = append(rows, []string{string(slot), filepath.Join(r.Dir, name)})
	}

	manifest := filepath.Join(under, r.Name+".tsv")
	written := &tsv.Table{
		Header: []tsv.ColumnName{project.SlotColumn, project.PathColumn},
		Rows:   rows,
	}
	if err := tsv.WriteFile(manifest, written); err != nil {
		return "", fmt.Errorf("wizard.Write: %w", err)
	}
	return manifest, nil
}
