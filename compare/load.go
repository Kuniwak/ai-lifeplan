package compare

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Sources struct {
	Root string

	ProjectPaths []string

	SlotOverrides map[tsv.Slot]string
}

func Load(sources Sources) ([]Subject, error) {
	if len(sources.ProjectPaths) < 2 {
		return nil, fmt.Errorf(
			"compare.Load: want at least two projects to compare, got %d", len(sources.ProjectPaths))
	}

	subjects := make([]Subject, 0, len(sources.ProjectPaths))
	seen := make(map[string]string, len(sources.ProjectPaths))
	for _, path := range sources.ProjectPaths {
		name := NameOf(path)
		if name == RecordSeries {
			return nil, fmt.Errorf(
				"compare.Load: %s は %q という名前だが、それは実績の系列の名前である。図の凡例でどちらか言えなくなるので、別の名前を付けること",
				path, RecordSeries)
		}
		if was, dup := seen[name]; dup {
			return nil, fmt.Errorf(
				"compare.Load: %s and %s are both named %q, so a column of the comparison could not say which it held",
				was, path, name)
		}
		seen[name] = path

		subject, err := loadOne(sources, path, name)
		if err != nil {
			return nil, err
		}
		subjects = append(subjects, subject)
	}
	return subjects, nil
}

func NameOf(manifestPath string) string {
	base := filepath.Base(manifestPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func loadOne(sources Sources, path, name string) (Subject, error) {
	in, err := plan.Load(plan.Sources{
		Root:          sources.Root,
		ProjectPath:   path,
		SlotOverrides: sources.SlotOverrides,
	})
	if err != nil {
		return Subject{}, err
	}
	paths := in.SlotPaths()

	startsAfter, err := in.StartsAfter()
	if err != nil {
		return Subject{}, err
	}
	built, err := in.Build()
	if err != nil {
		return Subject{}, err
	}

	read, err := in.Table(input.BalanceSlot)
	if err != nil {
		return Subject{}, err
	}
	record, err := actuals.ParseBalanceTable(read)
	if err != nil {
		return Subject{}, err
	}

	overridden := make(map[tsv.Slot]bool, len(sources.SlotOverrides))
	for slot := range sources.SlotOverrides {
		if _, fills := paths[slot]; fills {
			overridden[slot] = true
		}
	}

	return Subject{
		Name:          name,
		Paths:         paths,
		Overridden:    overridden,
		StartsAfter:   startsAfter,
		Record:        record,
		Tables:        built.Tables(),
		UnreadColumns: in.UnreadColumns(),
	}, nil
}

type Compared struct {
	Warnings []string

	Chart []byte

	slots          *tsv.Table
	timeline       *tsv.Table
	diff           *tsv.Table
	summary        *tsv.Table
	actualVsPlan   *tsv.Table
	spendingVsPlan *tsv.Table
}

func Of(subjects []Subject) (*Compared, error) {
	timeline, err := Timeline(subjects)
	if err != nil {
		return nil, err
	}
	diff, err := Diff(subjects)
	if err != nil {
		return nil, err
	}
	summary, err := Summary(subjects)
	if err != nil {
		return nil, err
	}
	actualVsPlan, err := ActualVsPlan(subjects)
	if err != nil {
		return nil, err
	}
	spendingVsPlan, err := SpendingVsPlan(subjects)
	if err != nil {
		return nil, err
	}
	lines, err := AssetsChart(subjects)
	if err != nil {
		return nil, err
	}
	drawn, err := lines.SVG()
	if err != nil {
		return nil, err
	}

	return &Compared{
		Warnings:       Warnings(subjects),
		Chart:          drawn,
		slots:          Slots(subjects),
		timeline:       timeline,
		diff:           diff,
		summary:        summary,
		actualVsPlan:   actualVsPlan,
		spendingVsPlan: spendingVsPlan,
	}, nil
}

type OutputName string

const (
	SlotsOutput    OutputName = "slots"
	TimelineOutput OutputName = "timeline"
	DiffOutput     OutputName = "diff"
	SummaryOutput  OutputName = "summary"

	ActualVsPlanOutput OutputName = "actual-vs-plan"

	SpendingVsPlanOutput OutputName = "spending-vs-plan"
)

func (c *Compared) Tables() map[OutputName]*tsv.Table {
	return map[OutputName]*tsv.Table{
		SlotsOutput:          c.slots,
		TimelineOutput:       c.timeline,
		DiffOutput:           c.diff,
		SummaryOutput:        c.summary,
		ActualVsPlanOutput:   c.actualVsPlan,
		SpendingVsPlanOutput: c.spendingVsPlan,
	}
}

func (c *Compared) WriteTables(dir string) error {
	for name, table := range c.Tables() {
		if err := tsv.WriteFile(filepath.Join(dir, string(name)+".tsv"), table); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compared) WriteChart(dir string) error {
	path := filepath.Join(dir, ChartFile)
	if err := os.WriteFile(path, c.Chart, 0o644); err != nil {
		return fmt.Errorf("compare: %s: %w", path, err)
	}
	return nil
}
