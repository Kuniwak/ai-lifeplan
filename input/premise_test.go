package input_test

import (
	"path/filepath"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/yeartest"
)

const (
	planStart date.Year = 2020
	planEnd   date.Year = 2090
)

const sheetsRounding money.Yen = 5_000

func TestEveryStepTableShouldAscendAndReachThePlanStart(t *testing.T) {
	for _, path := range []string{
		"../data/environment/residence.tsv",
		"../data/environment/investment-return.tsv",
		controllable("income-husband.tsv"),
		controllable("income-wife.tsv"),
		controllable("living-cost.tsv"),
		controllable("allowance-husband.tsv"),
		controllable("allowance-wife.tsv"),
		controllable("medical-expense.tsv"),
		controllable("life-insurance-premium.tsv"),
		controllable("housing-rent.tsv"),
		controllable("investment.tsv"),
	} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			table := read(t, path)

			years := yearsOf(t, table)
			for i := 1; i < len(years); i++ {
				if years[i] <= years[i-1] {
					t.Errorf("row %d has year %d, which does not come after %d", i+1, years[i], years[i-1])
				}
			}
			if len(years) == 0 {
				t.Fatal("no rows, so the plan's first year has no value")
			}
			if years[0] > planStart {
				t.Errorf("the first year written is %d, after the plan starts in %d", years[0], planStart)
			}
		})
	}
}

func TestEveryAmountShouldBeAWholeNumberOfYen(t *testing.T) {
	for path, columns := range map[string][]tsv.ColumnName{
		controllable("income-husband.tsv"):         {"給与収入[円/年]", "賞与収入[円/年]", "事業収入[円/年]", "事業必要経費[円/年]", "業務雑収入[円/年]"},
		controllable("income-wife.tsv"):            {"給与収入[円/年]", "賞与収入[円/年]", "事業収入[円/年]", "事業必要経費[円/年]"},
		controllable("tuition.tsv"):                {"学費[円/年]"},
		controllable("child-living-cost.tsv"):      {"生活費[円/年]"},
		controllable("living-cost.tsv"):            {"生活費[円/月]"},
		controllable("allowance-husband.tsv"):      {"小遣い[円/月]"},
		controllable("allowance-wife.tsv"):         {"小遣い[円/月]"},
		controllable("medical-expense.tsv"):        {"医療費[円]", "保険で補填された金額[円]"},
		controllable("extraordinary-cost.tsv"):     {"費用[円]"},
		controllable("life-insurance-premium.tsv"): {"生命保険料[円/年]"},
		controllable("property-insurance.tsv"):     {"火災保険料[円]", "地震保険料[円]"},
		controllable("housing.tsv"):                {"頭金[円]"},
		controllable("loan.tsv"):                   {"借入額[円]"},
		controllable("housing-rent.tsv"):           {"家賃[円/年]"},
		controllable("housing-maintenance.tsv"):    {"修繕費[円]"},
		controllable("investment.tsv"):             {"積立額[円/月]", "貯蓄維持目標[円]"},
	} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			table := read(t, path)
			for _, column := range columns {
				i, ok := table.ColumnIndex(column)
				if !ok {
					t.Errorf("the column %q is missing", column)
					continue
				}
				for row, fields := range table.Rows {
					if _, err := money.ParseYen(fields[i]); err != nil {
						t.Errorf("row %d, column %q: %v", row+1, column, err)
					}
				}
			}
		})
	}
}

func controllable(name string) string { return filepath.Join("..", "data", "controllable", name) }

func read(t *testing.T, path string) *tsv.Table {
	t.Helper()

	table, err := tsv.ReadFile(path)
	if err != nil {
		t.Fatalf("tsv.ReadFile: %v", err)
	}
	return table
}

func yearsOf(t *testing.T, table *tsv.Table) []date.Year {
	t.Helper()

	years := make([]date.Year, 0, len(table.Rows))
	yeartest.EachYear(t, table, func(year date.Year, _ []string) {
		years = append(years, year)
	})
	return years
}
