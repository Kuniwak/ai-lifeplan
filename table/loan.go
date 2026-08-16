package table

import (
	"fmt"
	"math"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type LoanName string

type Loan struct {
	Principal money.Yen

	AnnualRate money.Rate

	Years int

	Name LoanName

	DrawnIn date.Year

	FirstYear  date.Year
	FirstMonth int

	FixedYears int
}

type Settlement = *date.Year

type Payment struct {
	Number int

	Year, Month int

	Remaining int

	Balance money.Yen

	Interest, Repaid money.Yen
}

func (l Loan) MonthlyRate() money.Rate {
	return l.AnnualRate.Div(date.MonthsAYear)
}

func (l Loan) Instalments() int { return l.Years * date.MonthsAYear }

func (l Loan) MonthlyPayment() (money.Yen, error) {
	if l.Principal < 0 {
		return 0, fmt.Errorf("table.Loan: the principal is negative: %d", l.Principal)
	}
	if l.Years <= 0 {
		return 0, fmt.Errorf("table.Loan: the term is %d years, so there is nothing to repay over", l.Years)
	}

	return l.instalmentOver(l.Instalments())
}

func (l Loan) instalmentOver(instalments int) (money.Yen, error) {
	if instalments <= 0 {
		return 0, fmt.Errorf("table.Loan: 残りの回数が %d である", instalments)
	}

	n := float64(instalments)
	i := l.MonthlyRate().Float64()

	if i == 0 {
		return money.Yen(math.Floor(float64(l.Principal) / n)), nil
	}
	if i < 0 {
		return 0, fmt.Errorf("table.Loan: the interest rate is negative: %s", l.AnnualRate)
	}

	growth := math.Pow(1+i, n)
	return money.Yen(math.Floor(float64(l.Principal) * i * growth / (growth - 1))), nil
}

func (l Loan) calendarOf(k int) (date.Year, int) {
	return relation.YearMonthOf(relation.MonthsSince(l.FirstYear, l.FirstMonth) + k - 1)
}

func annualOf(monthly money.Rate) money.Rate {
	return money.NewRate(monthly.Num()*date.MonthsAYear, monthly.Den())
}

func (l Loan) Schedule(floating relation.Table[money.Rate]) ([]Payment, error) {
	instalment, err := l.MonthlyPayment()
	if err != nil {
		return nil, err
	}
	if l.FixedYears <= 0 || l.FixedYears > l.Years {
		return nil, fmt.Errorf(
			"table.Loan: 固定期間が %d 年で、返済期間の %d 年に収まらない。全期間固定なら %d と書く",
			l.FixedYears, l.Years, l.Years)
	}

	rate := l.MonthlyRate()
	total := l.Instalments()
	fixed := l.FixedYears * date.MonthsAYear

	schedule := make([]Payment, 0, total)
	balance := l.Principal
	charged := money.Rate{}
	for k := 1; k <= total; k++ {
		year, month := l.calendarOf(k)

		if k > fixed {
			nominal, ok := floating.At(year)
			if !ok {
				return nil, fmt.Errorf("table.Loan: %d の変動金利が分からない", year)
			}
			if nominal != charged {
				charged = nominal
				if instalment, err = (Loan{Principal: balance, AnnualRate: nominal}).
					instalmentOver(total - k + 1); err != nil {
					return nil, err
				}
				rate = nominal.Div(date.MonthsAYear)
			}
		}

		interest := balance.Mul(rate, money.Truncate)

		repaid := instalment - interest
		if repaid > balance || k == total {
			repaid = balance
		}

		schedule = append(schedule, Payment{
			Number:    k,
			Year:      int(year - l.FirstYear + 1),
			Month:     month,
			Remaining: total - k + 1,
			Balance:   balance,
			Interest:  interest,
			Repaid:    repaid,
		})
		balance -= repaid
	}

	return schedule, nil
}

type LoanYear struct {
	Paid money.Yen

	Interest, Repaid money.Yen

	Balance money.Yen
}

func NominalRatesByLoan(
	real map[LoanName]money.Rate, from, to date.Year, prices relation.Table[money.Rate],
) map[LoanName]relation.Table[money.Rate] {
	years := relation.Span(from, to)

	nominal := make(map[LoanName]relation.Table[money.Rate], len(real))
	for name, rate := range real {
		nominal[name] = NominalReturns(relation.Constant(years, rate), prices)
	}
	return nominal
}

func LoansTable(loans []Loan, settled Settlement, from, to date.Year, floating map[LoanName]relation.Table[money.Rate]) (relation.Table[LoanYear], error) {
	var empty relation.Table[LoanYear]

	total := make(map[date.Year]LoanYear, int(to-from)+1)
	for _, loan := range loans {
		rates, ok := floating[loan.Name]
		if !ok {
			return empty, fmt.Errorf("table.LoansTable: %q の変動金利が渡されていない", loan.Name)
		}
		built, err := loan.LoanTable(settled, from, to, rates)
		if err != nil {
			return empty, err
		}
		for _, row := range built.Rows() {
			year := total[row.Year]
			year.Paid += row.Value.Paid
			year.Interest += row.Value.Interest
			year.Repaid += row.Value.Repaid
			year.Balance += row.Value.Balance
			total[row.Year] = year
		}
	}

	return relation.Over(relation.Span(from, to),
		func(y date.Year) LoanYear { return total[y] }), nil
}

func (l Loan) LoanTable(settled Settlement, from, to date.Year, floating relation.Table[money.Rate]) (relation.Table[LoanYear], error) {
	var empty relation.Table[LoanYear]

	schedule, err := l.Schedule(floating)
	if err != nil {
		return empty, err
	}

	last, _ := l.calendarOf(l.Instalments())
	if settled != nil {
		if *settled < l.FirstYear || *settled > last {
			return empty, fmt.Errorf(
				"table.Loan: 一括返済の年 %d が返済期間の外にある。%s は %d 年から %d 年まで返す",
				*settled, l.Name, l.FirstYear, last)
		}
	}

	byYear := make(map[date.Year]LoanYear, l.Years)
	for _, p := range schedule {
		y := l.FirstYear + date.Year(p.Year) - 1

		if settled != nil && y > *settled {
			continue
		}

		year := byYear[y]
		year.Interest += p.Interest
		year.Repaid += p.Repaid
		year.Paid += p.Interest + p.Repaid
		year.Balance = p.Balance - p.Repaid
		byYear[y] = year
	}

	if settled != nil {
		year := byYear[*settled]
		year.Repaid += year.Balance
		year.Paid += year.Balance
		year.Balance = 0
		byYear[*settled] = year
	}

	for y := l.DrawnIn; y < l.FirstYear; y++ {
		if _, paid := byYear[y]; !paid {
			byYear[y] = LoanYear{Balance: l.Principal}
		}
	}

	return relation.Over(relation.Span(from, to),
		func(y date.Year) LoanYear { return byYear[y] }), nil
}
