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
	"github.com/Kuniwak/lifeplan/input"
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

// housingHeader and housingLines answer a question the cells cannot: the cells
// hold one number per plan at 2090, and the rent and the collateral are two
// series that move over the whole span. The rent is carried up by prices; the
// collateral is a figure read off a tax notice and stays where it is. How far
// apart they drift is set by the economy, so one plan per economy is enough,
// and the other conditions are left at the level the plan itself assumes.
func housingHeader() []string {
	return []string{"経済", "住まい", "西暦", "物価指数", "家賃", "担保評価額", "売却受取額", "売却後家賃"}
}

func axisNamed(axes []axis, name string) axis {
	for _, a := range axes {
		if a.name == name {
			return a
		}
	}
	panic("sweep: 軸 " + name + " が無い")
}

// collateralOf reads the land value the last resort pledges. plan.Build keeps
// it out of reach unless a plan actually runs short, so it is read here from
// the table the manifest names rather than taken off a plan that happened to
// need it.
func collateralOf(root string) (money.Yen, error) {
	loaded, err := plan.Load(plan.Sources{
		Root:        root,
		ProjectPath: filepath.Join(root, "projects", "base.tsv"),
	})
	if err != nil {
		return 0, err
	}
	path, ok := loaded.SlotPaths()[input.PropertyAssessmentSlot]
	if !ok {
		return 0, fmt.Errorf("sweep: %q を埋める表が無い", input.PropertyAssessmentSlot)
	}
	t, err := tsv.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return 0, err
	}
	r, err := tsv.NewReader(t, input.PropertyAssessmentSlot, input.LandValueColumn)
	if err != nil {
		return 0, err
	}
	return r.Yen(0, input.LandValueColumn)
}

// sellAndRentYearly is the rent the household would pay after selling, in the
// prices of the plan's first year. It is not the rent of the renting scenario:
// that one moves to a smaller flat once the children leave, and this one keeps
// the size the family had.
func sellAndRent(root string) (table.Measure, error) {
	loaded, err := plan.Load(plan.Sources{
		Root:        root,
		ProjectPath: filepath.Join(root, "projects", "base.tsv"),
	})
	if err != nil {
		return table.Measure{}, err
	}
	path, ok := loaded.SlotPaths()[input.LastResortSlot]
	if !ok {
		return table.Measure{}, fmt.Errorf("sweep: %q を埋める表が無い", input.LastResortSlot)
	}
	t, err := tsv.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return table.Measure{}, err
	}
	measures, err := table.Measures(t)
	if err != nil {
		return table.Measure{}, err
	}
	measure, ok := measures[table.SellAndRent]
	if !ok {
		return table.Measure{}, fmt.Errorf("sweep: %q が %q に無い", table.SellAndRent, input.LastResortSlot)
	}
	return measure, nil
}

func housingLines(root string, axes []axis) ([]string, error) {
	collateral, err := collateralOf(root)
	if err != nil {
		return nil, err
	}
	measure, err := sellAndRent(root)
	if err != nil {
		return nil, err
	}
	yearlyRent := measure.RentMonthly * date.MonthsAYear

	econ, housing := axisNamed(axes, "経済"), axisNamed(axes, "住まい")
	var lines []string
	for _, e := range econ.levels {
		for _, h := range housing.levels {
			overrides := map[tsv.Slot]string{}
			for slot, path := range e.slots {
				overrides[slot] = path
			}
			owns := h.key == "持ち家"
			for slot, path := range h.slots {
				overrides[slot] = path
			}
			built, err := plan.Build(plan.Sources{
				Root:          root,
				ProjectPath:   filepath.Join(root, "projects", "base.tsv"),
				SlotOverrides: overrides,
			})
			if err != nil {
				return nil, fmt.Errorf("sweep: %s / %s: %w", e.key, h.key, err)
			}
			pledged, proceeds := money.Yen(0), money.Yen(0)
			if owns {
				pledged, proceeds = collateral, measure.Proceeds(collateral)
			}
			for _, row := range built.Expense.Rows() {
				level, ok := built.PriceLevels.At(row.Year)
				if !ok {
					return nil, fmt.Errorf("sweep: %d 年の物価が無い", row.Year)
				}
				lines = append(lines, strings.Join([]string{
					e.key, h.key, fmt.Sprint(row.Year), level.String(),
					fmt.Sprint(int64(row.Value.Rent)), fmt.Sprint(int64(pledged)),
					fmt.Sprint(int64(proceeds)), fmt.Sprint(int64(level.Apply(yearlyRent))),
				}, "\t"))
			}
		}
	}
	return lines, nil
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
	housingOut := flag.String("housing-out", "out/sweep/housing.tsv", "write the rent and collateral series here")
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

	series, err := housingLines(*root, axes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	housingBody := strings.Join(housingHeader(), "\t") + "\n" + strings.Join(series, "\n") + "\n"
	if err := os.WriteFile(*housingOut, []byte(housingBody), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d rows to %s\n", len(series), *housingOut)
}
