package table_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/sheets"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

var theLoanOfTheBaseProject = table.Loan{
	Name:       "Sheets の一本",
	Principal:  50_000_000,
	AnnualRate: money.NewRate(2, 100),
	Years:      35,
	FixedYears: 20,
	DrawnIn:    2022,
	FirstYear:  2023,
	FirstMonth: 1,
}

func TestTheScheduleShouldClearTheLoanExactly(t *testing.T) {
	schedule, err := theLoanOfTheBaseProject.Schedule(theFloatingOfTheBaseProject(2018, 2090))
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if got, want := len(schedule), 420; got != want {
		t.Fatalf("%d instalments, want %d", got, want)
	}

	var repaid money.Yen
	for _, p := range schedule {
		if p.Balance < 0 {
			t.Fatalf("instalment %d: the balance went below nothing: %d", p.Number, p.Balance)
		}
		repaid += p.Repaid
	}

	if repaid != theLoanOfTheBaseProject.Principal {
		t.Errorf("repaid %d of %d", repaid, theLoanOfTheBaseProject.Principal)
	}
	if last := schedule[len(schedule)-1]; last.Balance-last.Repaid != 0 {
		t.Errorf("the loan ends owing %d", last.Balance-last.Repaid)
	}
}

func TestTheYearlyLoanTableShouldCoverThePlanEvenBeforeAndAfterTheLoan(t *testing.T) {
	built, err := theLoanOfTheBaseProject.LoanTable(nil, 2018, 2090, theFloatingOfTheBaseProject(2018, 2090))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	if got, want := built.Len(), 73; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}

	for year, want := range map[date.Year]money.Yen{
		2018: 0,
		2022: 0,
		2090: 0,
	} {
		row, _ := built.At(year)
		if row.Paid != want {
			t.Errorf("%d: paid %d, want %d", year, row.Paid, want)
		}
	}

	for _, c := range []struct {
		year date.Year
		want money.Yen
	}{
		{2023, 1_990_000},
		{2040, 1_990_000},
		{2043, 2_070_000},
		{2057, 2_070_000},
	} {
		row, _ := built.At(c.year)
		if diff := row.Paid - c.want; diff > 10_000 || diff < -10_000 {
			t.Errorf("%d: paid %d, want about %d", c.year, row.Paid, c.want)
		}
	}
	if row, _ := built.At(date.Year(2058)); row.Paid != 0 {
		t.Errorf("2058: paid %d, want nothing; the term ends in 2057", row.Paid)
	}
}

func TestTheYearEndBalanceShouldFallToNothingByTheEndOfTheTerm(t *testing.T) {
	built, err := theLoanOfTheBaseProject.LoanTable(nil, 2018, 2090, theFloatingOfTheBaseProject(2018, 2090))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	if row, _ := built.At(date.Year(2057)); row.Balance != 0 {
		t.Errorf("2057: %d still owed at the end of the term", row.Balance)
	}

	previous := money.Yen(-1)
	for _, row := range built.Rows() {
		if row.Year < theLoanOfTheBaseProject.FirstYear {
			continue
		}
		if previous >= 0 && row.Value.Balance > previous {
			t.Errorf("%d: the balance rose to %d from %d", row.Year, row.Value.Balance, previous)
		}
		previous = row.Value.Balance
	}
}

func manyen(t *testing.T, golden *tsv.Table, fields []string, column tsv.ColumnName, unit string) money.Yen {
	t.Helper()

	v, err := sheets.ManYen(strings.TrimSuffix(fields[columnIndex(t, golden, column)], unit))
	if err != nil {
		t.Fatalf("column %q: %v", column, err)
	}
	return v
}

func count(t *testing.T, golden *tsv.Table, fields []string, column tsv.ColumnName, unit string) int {
	t.Helper()

	field := strings.TrimSuffix(fields[columnIndex(t, golden, column)], unit)
	n, err := strconv.Atoi(field)
	if err != nil {
		t.Fatalf("column %q: %v", column, err)
	}
	return n
}

func TestTheWholeLoanShouldBeOwedAtTheEndOfTheYearItWasDrawnIn(t *testing.T) {
	built, err := theLoanOfTheBaseProject.LoanTable(nil, 2018, 2090, theFloatingOfTheBaseProject(2018, 2090))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	row, _ := built.At(date.Year(2022))
	if row.Balance != theLoanOfTheBaseProject.Principal {
		t.Errorf("2022 balance = %d, want the whole principal %d", row.Balance, theLoanOfTheBaseProject.Principal)
	}
	if row.Paid != 0 {
		t.Errorf("2022 paid = %d, want nothing", row.Paid)
	}

	if row, _ := built.At(date.Year(2021)); row.Balance != 0 {
		t.Errorf("2021 balance = %d, want nothing; the home had not been bought", row.Balance)
	}
}

func TestTheYearEndBalanceShouldSitCloseToTheTaxReturns(t *testing.T) {
	built, err := theLoanOfTheBaseProject.LoanTable(nil, 2018, 2090, theFloatingOfTheBaseProject(2018, 2090))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	for year, want := range map[date.Year]money.Yen{
		2024: 47_732_000,
		2025: 46_687_000,
	} {
		row, _ := built.At(year)
		within := want * 6 / 1000
		if diff := row.Balance - want; diff > within || diff < -within {
			t.Errorf("%d: balance = %d, the 確定申告書 says %d (differs by %d, more than the %d the drawdown explains)",
				year, row.Balance, want, diff, within)
		}
	}
}

func TestALoanSettledInAYearShouldBeClearedAtTheEndOfIt(t *testing.T) {
	settled := date.Year(2032)
	loan := table.Loan{
		Principal: 50_000_000, AnnualRate: money.NewRate(2, 100),
		Years: 35, FixedYears: 35, FirstYear: 2023, FirstMonth: 1,
	}

	built, err := loan.LoanTable(&settled, 2022, 2060, noInflation(2022, 2060))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	at := func(y date.Year) table.LoanYear {
		t.Helper()
		row, ok := built.At(y)
		if !ok {
			t.Fatalf("%d の行が無い", y)
		}
		return row
	}
	before, year, after := at(settled-1), at(settled), at(settled+1)

	if year.Balance != 0 {
		t.Errorf("一括返済した年の年末残高が %d 残っている", year.Balance)
	}
	if want := before.Balance + year.Interest; year.Paid != want {
		t.Errorf("一括返済した年の返済額 %d が、前年末残高 %d ＋ 利息 %d = %d と違う",
			year.Paid, before.Balance, year.Interest, want)
	}
	if after.Paid != 0 || after.Balance != 0 {
		t.Errorf("一括返済の翌年に返済 %d / 残高 %d が残っている", after.Paid, after.Balance)
	}
}

func TestALoanWithNoSettlementYearShouldRunToTheEndOfItsTerm(t *testing.T) {
	loan := table.Loan{
		Principal: 50_000_000, AnnualRate: money.NewRate(2, 100),
		Years: 35, FixedYears: 35, FirstYear: 2023, FirstMonth: 1,
	}

	built, err := loan.LoanTable(nil, 2022, 2060, noInflation(2022, 2060))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	last, ok := built.At(2057)
	if !ok {
		t.Fatal("2057 の行が無い")
	}
	if last.Balance != 0 || last.Paid == 0 {
		t.Errorf("35 年で終わっていない: 2057 年の返済 %d / 残高 %d", last.Paid, last.Balance)
	}
}

func TestALoanSettledInTheYearItEndsShouldDoNothing(t *testing.T) {
	settled := date.Year(2057)
	loan := table.Loan{
		Principal: 50_000_000, AnnualRate: money.NewRate(2, 100),
		Years: 35, FixedYears: 35, FirstYear: 2023, FirstMonth: 1,
	}

	a, err := loan.LoanTable(&settled, 2022, 2060, noInflation(2022, 2060))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}
	b, err := loan.LoanTable(nil, 2022, 2060, noInflation(2022, 2060))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	for _, row := range b.Rows() {
		got, _ := a.At(row.Year)
		if got != row.Value {
			t.Errorf("%d: 完済の年に一括返済を書くと %+v が %+v に変わった", row.Year, row.Value, got)
		}
	}
}

func TestALoanSettledOutsideItsTermShouldBeRefused(t *testing.T) {
	for _, c := range []struct {
		name    string
		settled date.Year
	}{
		{"借りる前", 2021},
		{"取得年と同じ", 2022},
		{"完済の翌年", 2058},
	} {
		t.Run(c.name, func(t *testing.T) {
			settled := c.settled
			loan := table.Loan{
				Principal: 50_000_000, AnnualRate: money.NewRate(2, 100),
				Years: 35, FixedYears: 35, FirstYear: 2023, FirstMonth: 1,
			}
			if _, err := loan.LoanTable(&settled, 2022, 2060, noInflation(2022, 2060)); err == nil {
				t.Errorf("%d 年の一括返済を拒んでいない（取得 2022 年、35 年）", c.settled)
			}
		})
	}
}

func theFloatingOfTheBaseProject(from, to date.Year) relation.Table[money.Rate] {
	return floatingAt(money.NewRate(2557, 100_000), from, to)
}

func noInflation(from, to date.Year) relation.Table[money.Rate] {
	return floatingAt(money.NewRate(0, 100), from, to)
}

func floatingAt(rate money.Rate, from, to date.Year) relation.Table[money.Rate] {
	const only table.LoanName = "契約"
	flat := relation.Constant(relation.Span(from, to), money.NewRate(0, 100))
	return table.NominalRatesByLoan(map[table.LoanName]money.Rate{only: rate}, from, to, flat)[only]
}

func TestAFloatingLoanShouldCostMoreAtAHigherRate(t *testing.T) {
	loan := table.Loan{
		Principal: 50_000_000, AnnualRate: money.NewRate(2, 100),
		Years: 35, FixedYears: 20, FirstYear: 2023, FirstMonth: 1,
	}

	low, err := loan.LoanTable(nil, 2022, 2060, floatingAt(money.NewRate(1, 100), 2022, 2060))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}
	high, err := loan.LoanTable(nil, 2022, 2060, floatingAt(money.NewRate(3, 100), 2022, 2060))
	if err != nil {
		t.Fatalf("LoanTable: %v", err)
	}

	a, _ := low.At(2045)
	b, _ := high.At(2045)
	if b.Interest <= a.Interest {
		t.Errorf("金利が 1%% と 3%% で利息が %d と %d、増えていない", a.Interest, b.Interest)
	}
}

func TestALoanWithNoFixedPeriodShouldBeRefused(t *testing.T) {
	for _, c := range []struct {
		name  string
		fixed int
	}{
		{"未記入", 0},
		{"負", -1},
		{"返済期間より長い", 36},
	} {
		t.Run(c.name, func(t *testing.T) {
			loan := table.Loan{
				Principal: 50_000_000, AnnualRate: money.NewRate(2, 100),
				Years: 35, FixedYears: c.fixed, FirstYear: 2023, FirstMonth: 1,
			}
			if _, err := loan.LoanTable(nil, 2022, 2060, noInflation(2022, 2060)); err == nil {
				t.Errorf("固定期間 %d 年を拒んでいない", c.fixed)
			}
		})
	}
}
