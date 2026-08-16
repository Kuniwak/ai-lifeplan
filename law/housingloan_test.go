package law_test

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/tsv"
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

const movedInIn = 2022

func housingLoanTable(t *testing.T) law.HousingLoanCreditTable {
	t.Helper()

	table := law.MustLoadHousingLoanCredits(t, os.DirFS("../"+law.LawDirectory))
	return table
}

func TestTheHousingLoanCreditShouldStopAfterItsYears(t *testing.T) {
	table := housingLoanTable(t)

	if got := table.Credit(36_000_000, 10_000_000, movedInIn, 2034); got <= 0 {
		t.Errorf("2034 credit = %d, want something; the thirteenth year is still in", got)
	}
	if got := table.Credit(35_000_000, 10_000_000, movedInIn, 2035); got != 0 {
		t.Errorf("2035 credit = %d, want nothing; the thirteen years are up", got)
	}
	if got := table.Credit(50_000_000, 10_000_000, movedInIn, 2021); got != 0 {
		t.Errorf("2021 credit = %d, want nothing; the home had not been moved into", got)
	}
}

func TestTheHousingLoanCreditShouldStopAtTheBorrowingCeiling(t *testing.T) {
	table := housingLoanTable(t)

	atCeiling := table.Credit(50_000_000, 10_000_000, movedInIn, 2023)
	above := table.Credit(90_000_000, 10_000_000, movedInIn, 2023)
	if atCeiling != above {
		t.Errorf("a balance of 90,000,000 earns %d against %d at the ceiling", above, atCeiling)
	}
	if want := money.Yen(350_000); atCeiling != want {
		t.Errorf("credit at the ceiling = %d, want %d", atCeiling, want)
	}
}

func TestTheHousingLoanCreditShouldLapseForAYearOfHighIncomeAndComeBack(t *testing.T) {
	table := housingLoanTable(t)

	if got := table.Credit(45_000_000, 25_000_000, movedInIn, 2027); got != 0 {
		t.Errorf("credit = %d in a year over the income limit, want nothing", got)
	}
	if got := table.Credit(45_000_000, 10_000_000, movedInIn, 2028); got <= 0 {
		t.Errorf("credit = %d in the year after, want it back", got)
	}
}

func TestTheCreditShouldNotReachPastItsRowsLastYear(t *testing.T) {
	shipped := housingLoanTable(t)

	for name, tc := range map[string]struct {
		movedIn date.Year
		want    bool
	}{
		"表の最後の年は当たる": {movedIn: 2025, want: true},
		"その翌年は当たらない": {movedIn: 2026, want: false},
		"はるか先も当たらない": {movedIn: 2090, want: false},
		"表より前も当たらない": {movedIn: 2021, want: false},
	} {
		t.Run(name, func(t *testing.T) {

			terms, ok := shipped.Terms(tc.movedIn)

			if ok != tc.want {
				t.Errorf("Terms(%d) ok=%v (%v のはず) terms=%+v", tc.movedIn, ok, tc.want, terms)
			}
		})
	}

	if got := shipped.Credit(50_000_000, 5_000_000, 2026, 2027); got != 0 {
		t.Errorf("2026 年入居に控除 %d 円が出ている", got)
	}
}

func TestTheCreditShouldRefuseTwoRowsClaimingTheSameYear(t *testing.T) {
	const header = "居住開始年\t控除率\t控除期間[年]\t借入限度額[円]\t合計所得金額上限[円]\t適用終了年\t出典\n"

	for name, written := range map[string]string{
		"無期限の行が期限つきの行にかぶる": header +
			"2010\t1.00%\t10\t40000000\t30000000\t無期限\tx\n" +
			"2022\t0.70%\t13\t50000000\t20000000\t2025\tx\n",
		"前の行の期限が次の行に食い込む": header +
			"2010\t1.00%\t10\t40000000\t30000000\t2050\tx\n" +
			"2022\t0.70%\t13\t50000000\t20000000\t2025\tx\n",
	} {
		t.Run(name, func(t *testing.T) {
			read, err := tsv.Read(strings.NewReader(written))
			if err != nil {
				t.Fatalf("tsv.Read: %v", err)
			}

			_, err = law.ParseHousingLoanCreditTable(read)

			if err == nil {
				t.Fatal("2 つの行が同じ年を名乗っているのに、表が組み上がってしまった")
			}
			if !strings.Contains(err.Error(), "2022") {
				t.Errorf("言われたのは %q だが、どの年が決まらないのかを言ってほしい", err)
			}
		})
	}
}

func TestTheCreditShouldNotAnswerTheYearsBetweenTwoRows(t *testing.T) {
	read, err := tsv.Read(strings.NewReader(
		"居住開始年\t控除率\t控除期間[年]\t借入限度額[円]\t合計所得金額上限[円]\t適用終了年\t出典\n" +
			"2010\t1.00%\t10\t40000000\t30000000\t2020\tx\n" +
			"2022\t0.70%\t13\t50000000\t20000000\t2025\tx\n"))
	if err != nil {
		t.Fatalf("tsv.Read: %v", err)
	}
	table, err := law.ParseHousingLoanCreditTable(read)
	if err != nil {
		t.Fatalf("law.ParseHousingLoanCreditTable: %v", err)
	}

	for name, tc := range map[string]struct {
		movedIn date.Year
		want    bool
	}{
		"前の行の最後の年（境界値）": {movedIn: 2020, want: true},
		"穴の年": {movedIn: 2021, want: false},
		"次の行の最初の年（境界値）": {movedIn: 2022, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			terms, ok := table.Terms(tc.movedIn)

			if ok != tc.want {
				t.Errorf("Terms(%d) ok=%v (%v のはず) terms=%+v", tc.movedIn, ok, tc.want, terms)
			}
		})
	}
}
