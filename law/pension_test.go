package law

import (
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
)

func TestPensionIncomeShouldMatchTheSpreadsheet(t *testing.T) {
	type testCase struct {
		Block string
		Age   int
	}

	testCases := map[string]testCase{
		"under 65":    {Block: "pension-income-deduction-under65", Age: 64},
		"65 and over": {Block: "pension-income-deduction-over65", Age: 65},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			table, err := sheets.New(os.DirFS("../testdata/sheets")).Table(tc.Block)
			if err != nil {
				t.Fatalf("Table: %v", err)
			}
			receivedColumn, ok := table.ColumnIndex("年金収入")
			if !ok {
				t.Fatal("no 年金収入 column")
			}
			incomeColumn, ok := table.ColumnIndex("年金所得額")
			if !ok {
				t.Fatal("no 年金所得額 column")
			}

			checked := 0
			for i, row := range table.Rows {
				received, err := parseManYen(row[receivedColumn])
				if err != nil {
					t.Fatalf("row %d: 年金収入: %v", i+1, err)
				}
				want, err := parseManYen(row[incomeColumn])
				if err != nil {
					t.Fatalf("row %d: 年金所得額: %v", i+1, err)
				}

				if got := PensionIncome(received, tc.Age, 0, 2024); got != want {
					t.Errorf("年金収入 %s万 at age %d: income %d, the spreadsheet says %d",
						row[receivedColumn], tc.Age, got, want)
				}
				checked++
			}

			if checked < 480 {
				t.Errorf("only %d rows were checked; the golden table looks truncated", checked)
			}
		})
	}
}

func drawIncomeYear(t *rapid.T) date.Year {
	return rapid.SampledFrom([]date.Year{2015, 2019, 2020, 2024, 2100}).Draw(t, "所得の年")
}

func TestTurning65ShouldNeverRaiseTheTaxedIncome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		received := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "received"))
		year := drawIncomeYear(t)

		under, over := PensionIncome(received, 64, 0, year), PensionIncome(received, 65, 0, year)

		if over > under {
			t.Fatalf("pension %d gives income %d at 64 but %d at 65", received, under, over)
		}
	})
}

func TestPensionIncomeShouldRiseWithTheAmountReceived(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "a"))
		b := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "b"))
		if a > b {
			a, b = b, a
		}
		age := rapid.IntRange(60, 100).Draw(t, "age")
		totalIncome := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "totalIncome"))
		year := drawIncomeYear(t)

		gotA, gotB := PensionIncome(a, age, totalIncome, year), PensionIncome(b, age, totalIncome, year)

		if gotA > gotB {
			t.Fatalf("pension %d gives income %d but the larger %d gives only %d", a, gotA, b, gotB)
		}
	})
}

func TestAHigherTotalIncomeShouldNeverLowerTheTaxedPension(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		received := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "received"))
		age := rapid.IntRange(60, 100).Draw(t, "age")
		low := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "low"))
		high := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "high"))
		if low > high {
			low, high = high, low
		}
		year := drawIncomeYear(t)

		gotLow, gotHigh := PensionIncome(received, age, low, year), PensionIncome(received, age, high, year)

		if gotHigh < gotLow {
			t.Fatalf("a total income of %d gives pension income %d, but the larger %d gives only %d",
				low, gotLow, high, gotHigh)
		}
	})
}

func TestPensionIncomeShouldNeverExceedWhatWasReceived(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		received := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "received"))
		age := rapid.IntRange(60, 100).Draw(t, "age")
		totalIncome := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "totalIncome"))
		year := drawIncomeYear(t)

		got := PensionIncome(received, age, totalIncome, year)

		if got > received {
			t.Fatalf("pension %d gives income %d, which is more than was received", received, got)
		}
		if got < 0 {
			t.Fatalf("pension %d gives a negative income %d", received, got)
		}
	})
}
