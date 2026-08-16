package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

type level struct {
	key   string
	slots map[tsv.Slot]string
}

type axis struct {
	name       string
	controlled bool
	levels     []level
}

func env(economy, pension string) map[tsv.Slot]string {
	return map[tsv.Slot]string{
		"inflation":         "data/environment/scenario/inflation-" + economy + ".tsv",
		"real_wage_growth":  "data/environment/scenario/wage-" + economy + ".tsv",
		"investment_return": "data/environment/scenario/return-" + economy + ".tsv",
		"pension_level":     "data/environment/scenario/pension-" + pension + ".tsv",
	}
}

func one(slot tsv.Slot, path string) map[tsv.Slot]string {
	return map[tsv.Slot]string{slot: path}
}

func Axes() []axis {
	return []axis{
		{"経済", false, []level{
			{"成長型", nil},
			{"高成長", env("high-growth", "high-growth")},
			{"過去30年", env("past30", "past30")},
			{"ゼロ成長", env("zero-growth", "zero-growth")},
			{"ゼロ成長枯渇", env("zero-growth", "zero-growth-depleted")},
			{"OECD", env("oecd", "growth-transition")},
			{"OECD低", env("oecd-low", "growth-transition")},
		}},
		{"生活費", true, []level{
			{"家計調査どおり", nil},
			{"−4万", one("living_cost", "data/controllable/scenario/living-cost-24.tsv")},
			{"+4万", one("living_cost", "data/controllable/scenario/living-cost-32.tsv")},
		}},
		{"年金受給開始", true, []level{
			{"現行(夫70妻66)", nil},
			{"65歳", one("pension", "data/controllable/scenario/pension-at-65.tsv")},
			{"70歳", one("pension", "data/controllable/scenario/pension-at-70.tsv")},
			{"75歳", one("pension", "data/controllable/scenario/pension-at-75.tsv")},
		}},
		{"住まい", true, []level{
			{"持ち家", nil},
			{"賃貸のまま", map[tsv.Slot]string{
				"housing":             "data/controllable/scenario/housing-never-bought.tsv",
				"loan":                "data/controllable/scenario/loan-none.tsv",
				"housing_rent":        "data/controllable/scenario/housing-rent-forever.tsv",
				"housing_maintenance": "data/controllable/scenario/housing-maintenance-none.tsv",
			}},
		}},
		{"金融危機", false, []level{
			{"なし", nil},
			{"2030年", one("financial_crisis", "data/environment/scenario/crisis-2030.tsv")},
			{"2040年", one("financial_crisis", "data/environment/scenario/crisis-2040.tsv")},
			{"2050年", one("financial_crisis", "data/environment/scenario/crisis-2050.tsv")},
			{"2060年", one("financial_crisis", "data/environment/scenario/crisis-2060.tsv")},
			{"2070年", one("financial_crisis", "data/environment/scenario/crisis-2070.tsv")},
			{"2050-2051年の2年", one("financial_crisis", "data/environment/scenario/crisis-2050-2year.tsv")},
		}},
	}
}

func Cells(axes []axis) [][]int {
	cells := [][]int{{}}
	for _, a := range axes {
		next := make([][]int, 0, len(cells)*len(a.levels))
		for _, c := range cells {
			for i := range a.levels {
				next = append(next, append(append([]int{}, c...), i))
			}
		}
		cells = next
	}
	return cells
}

type result struct {
	assets    map[date.Year]money.Yen
	available map[date.Year]money.Yen
	shortfall map[date.Year]money.Yen
	ruin      date.Year

	resort     table.MeasureName
	resortFrom date.Year
}

var years = []date.Year{2090}

func run(root string, axes []axis, cell []int) (result, error) {
	overrides := map[tsv.Slot]string{}
	for i, a := range axes {
		for slot, path := range a.levels[cell[i]].slots {
			overrides[slot] = path
		}
	}
	built, err := plan.Build(plan.Sources{
		Root:          root,
		ProjectPath:   filepath.Join(root, "projects", "base.tsv"),
		SlotOverrides: overrides,
	})
	if err != nil {
		return result{}, err
	}

	out := result{
		resort:     built.LastResort.Measure.Name,
		resortFrom: built.LastResort.From,
		assets:     map[date.Year]money.Yen{},
		available:  map[date.Year]money.Yen{},
		shortfall:  map[date.Year]money.Yen{},
	}
	var short money.Yen
	for _, row := range built.Assets.Rows() {
		level, ok := built.PriceLevels.At(row.Year)
		if !ok {
			return result{}, fmt.Errorf("sweep: %d 年の物価が無い", row.Year)
		}
		if out.ruin == 0 && row.Value.Shortfall > 0 {
			out.ruin = row.Year
		}
		short += level.Deflate(row.Value.Shortfall)
		for _, year := range years {
			if row.Year == year {
				out.assets[year] = level.Deflate(row.Value.Total)
				out.available[year] = level.Deflate(row.Value.Available())
				out.shortfall[year] = short
			}
		}
	}
	for _, year := range years {
		if _, ok := out.assets[year]; !ok {
			return result{}, fmt.Errorf("sweep: %d 年の資産が無い", year)
		}
	}
	return out, nil
}

func header(axes []axis) []string {
	head := make([]string, 0, len(axes)+4*len(years)+1)
	for _, a := range axes {
		head = append(head, a.name)
	}
	for _, year := range years {
		head = append(head, fmt.Sprintf("資産%d", year), fmt.Sprintf("手が届く額%d", year),
			fmt.Sprintf("累積不足%d", year), fmt.Sprintf("純資産%d", year))
	}
	return append(head, "破産年", "最後の手段", "最後の手段の年")
}

func line(axes []axis, cell []int, r result) []string {
	fields := make([]string, 0, len(axes)+4*len(years)+1)
	for i, a := range axes {
		fields = append(fields, a.levels[cell[i]].key)
	}
	for _, year := range years {
		fields = append(fields,
			fmt.Sprint(int64(r.assets[year])), fmt.Sprint(int64(r.available[year])),
			fmt.Sprint(int64(r.shortfall[year])),
			fmt.Sprint(int64(r.available[year]-r.shortfall[year])))
	}
	ruin, resortFrom := "", ""
	if r.ruin != 0 {
		ruin = fmt.Sprint(r.ruin)
	}
	if r.resortFrom != 0 {
		resortFrom = fmt.Sprint(r.resortFrom)
	}
	return append(fields, ruin, string(r.resort), resortFrom)
}

func main() {
	root := flag.String("root", ".", "take the paths written in the manifest from here")
	out := flag.String("out", "out/sweep/cells.tsv", "write the cells here")
	flag.Parse()

	axes := Axes()
	cells := Cells(axes)

	lines := make([]string, len(cells))
	var failed sync.Map
	var wg sync.WaitGroup
	work := make(chan int)
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				r, err := run(*root, axes, cells[i])
				if err != nil {
					failed.Store(i, err)
					continue
				}
				lines[i] = strings.Join(line(axes, cells[i], r), "\t")
			}
		}()
	}
	for i := range cells {
		work <- i
	}
	close(work)
	wg.Wait()

	var errs []string
	failed.Range(func(i, err any) bool {
		errs = append(errs, fmt.Sprintf("cell %d: %v", i, err))
		return true
	})
	if len(errs) > 0 {
		sort.Strings(errs)
		fmt.Fprintln(os.Stderr, "Error:", strings.Join(errs[:min(len(errs), 5)], "\n  "))
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	body := strings.Join(header(axes), "\t") + "\n" + strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(*out, []byte(body), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d cells to %s\n", len(cells), *out)
}
