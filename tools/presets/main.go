package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type slot struct{ name, path string }

type preset struct {
	key   string
	label string
	slots []slot
}

var economies = []preset{
	{"growth", "財政検証 成長型経済移行・継続", envSlots("growth-transition", "growth-transition")},
	{"high-growth", "財政検証 高成長実現", envSlots("high-growth", "high-growth")},
	{"past30", "財政検証 過去30年投影", envSlots("past30", "past30")},
	{"zero-growth", "財政検証 1人当たりゼロ成長", envSlots("zero-growth", "zero-growth")},
	{"zero-growth-depleted", "ゼロ成長＋年金調整が長引く", envSlots("zero-growth", "zero-growth-depleted")},
	{"oecd", "OECD base case", envSlots("oecd", "growth-transition")},
	{"oecd-low", "OECD alternative", envSlots("oecd-low", "growth-transition")},
}

func envSlots(economy, pension string) []slot {
	return []slot{
		{"inflation", "data/environment/scenario/inflation-" + economy + ".tsv"},
		{"real_wage_growth", "data/environment/scenario/wage-" + economy + ".tsv"},
		{"investment_return", "data/environment/scenario/return-" + economy + ".tsv"},
		{"pension_level", "data/environment/scenario/pension-" + pension + ".tsv"},
	}
}

var repairs = []preset{
	{"repair-mid", "修繕費 +0.541pt（1990→2025 の窓）", nil},
	{"repair-low", "修繕費 +0.262pt（1970→2025 の窓）",
		[]slot{{"inflation_target", "data/environment/scenario/repair-low.tsv"}}},
	{"repair-high", "修繕費 +1.428pt（最も急な窓）",
		[]slot{{"inflation_target", "data/environment/scenario/repair-high.tsv"}}},
}

var dials = []preset{
	{"best", "最良執行", nil},
	{"as-now", "現状のまま（給与の下り坂・積立 10 万・課税から先に売る）", []slot{
		{"income_husband", "data/controllable/income-husband.tsv"},
		{"investment", "data/controllable/investment.tsv"}}},
	{"invest-as-now", "運用だけ現状", []slot{
		{"investment", "data/controllable/investment.tsv"}}},
	{"income-decline", "給与だけ下り坂", []slot{
		{"income_husband", "data/controllable/income-husband.tsv"}}},
	{"income-flat", "給与を定年まで維持", []slot{
		{"income_husband", "data/controllable/scenario/income-husband-flat-to-retirement.tsv"}}},
	{"settle-2050", "2042 年にローンを一括返済", []slot{
		{"loan_settlement", "data/controllable/scenario/loan-settlement-2050.tsv"}}},
}

var livings = []preset{
	{"living-30", "生活費 30 万円/月（実績のよこびき）", nil},
	{"living-26", "生活費 26 万円/月", []slot{
		{"living_cost", "data/controllable/scenario/living-cost-24.tsv"}}},
	{"living-34", "生活費 34 万円/月", []slot{
		{"living_cost", "data/controllable/scenario/living-cost-32.tsv"}}},
}

func Environments() []preset { return cross(economies, repairs) }
func Dials() []preset        { return cross(dials, livings) }

func cross(a, b []preset) []preset {
	out := make([]preset, 0, len(a)*len(b))
	for _, x := range a {
		for _, y := range b {
			out = append(out, preset{
				key:   x.key + "--" + y.key,
				label: x.label + " ／ " + y.label,
				slots: append(append([]slot{}, x.slots...), y.slots...),
			})
		}
	}
	return out
}

func Files() map[string]string {
	files := make(map[string]string)
	for _, env := range Environments() {
		files[filepath.Join("env", env.key+".tsv")] =
			manifest("../base.tsv", env.label, env.slots)
	}
	for _, env := range Environments() {
		for _, dial := range Dials() {
			files[filepath.Join("product", env.key+"--"+dial.key+".tsv")] =
				manifest("../env/"+env.key+".tsv", env.label+" ／ "+dial.label, dial.slots)
		}
	}
	return files
}

func manifest(parent, _ string, slots []slot) string {
	var b strings.Builder
	b.WriteString("slot\tpath\n")
	fmt.Fprintf(&b, "extends\t%s\n", parent)
	for _, s := range slots {
		fmt.Fprintf(&b, "%s\t%s\n", s.name, s.path)
	}
	return b.String()
}

func main() {
	out := flag.String("out", "projects", "where the manifests are written")
	flag.Parse()

	files := Files()
	for _, dir := range []string{"env", "product"} {
		if err := os.RemoveAll(filepath.Join(*out, dir)); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Join(*out, dir), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(*out, name), []byte(files[name]), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
	fmt.Printf("wrote %d manifests: 環境 %d × ダイヤル %d\n",
		len(files), len(Environments()), len(Dials()))
}
