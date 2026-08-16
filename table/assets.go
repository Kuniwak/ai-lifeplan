package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

func (r AssetsRow) Available() money.Yen { return r.Cash + r.Invested }

func (r AssetsRow) UnrealisedGain() money.Yen { return r.Taxable - r.Basis }

func (r AssetsRow) DeferredTax() money.Yen {
	return law.InvestmentIncomeTaxOn(r.UnrealisedGain())
}

type AssetsRow struct {
	Balance money.Yen

	Cash, Invested money.Yen

	Locked money.Yen

	Contributed, Withdrawn money.Yen

	Returns, Crash money.Yen

	NISA, Taxable, Basis money.Yen

	MaturedFromNISA money.Yen

	InvestmentTax money.Yen

	Paid money.Yen

	PensionReceived, PensionTax money.Yen

	Raised, Interest, Housing money.Yen

	Shortfall money.Yen

	Total money.Yen

	Recorded bool
}

func PensionReceipt(
	calendar relation.Table[CalendarRow],
	person PersonName,
	contribution relation.Table[money.Yen],
) (*date.Year, int) {
	var drawable *date.Year
	for _, row := range calendar.Rows() {
		age, ok := row.Value.AgeOf(person)
		if !ok || age < law.PensionDrawableAge {
			continue
		}
		year := row.Year
		drawable = &year
		break
	}
	if drawable == nil {
		return nil, 0
	}

	years, paidIn := 0, false
	var stopped *date.Year
	for _, row := range contribution.Rows() {
		switch {
		case row.Value > 0:
			years++
			paidIn = true
			stopped = nil
		case paidIn && stopped == nil:
			year := row.Year
			stopped = &year
		}
	}
	if !paidIn {
		return nil, 0
	}

	received := *drawable
	if stopped != nil && *stopped > received {
		received = *stopped
	}
	return &received, years
}

type AssetsInput struct {
	Timeline relation.Table[TimelineRow]

	Opening     Balance
	StartsAfter date.Year

	Actual map[date.Year]Balance

	ContributionMonthly, CashFloor relation.Table[money.Yen]

	MaturityOfOldNISA date.Year

	LastResort LastResort

	SellNISAFirst bool

	NISAAllowance relation.Table[money.Yen]

	Return relation.Table[money.Rate]

	Crash map[date.Year]money.Rate

	PensionReceivedIn   *date.Year
	PensionServiceYears int

	ResidentTax law.ResidentRates

	MutualAid relation.Table[money.Yen]
}

type Balance = actuals.Balance

func shortfallOf(cash money.Yen) money.Yen {
	if cash >= 0 {
		panic(fmt.Sprintf(
			"table.shortfallOf: 残高 %d は負ではない。不足は「取り崩しても払えなかった額」なので、"+
				"払えている年に立ててはいけない", cash))
	}
	return -cash
}

func AssetsTable(in AssetsInput) (relation.Table[AssetsRow], error) {
	var empty relation.Table[AssetsRow]

	years := in.Timeline.Years()
	rows := make([]relation.Row[AssetsRow], 0, len(years))

	cash, locked := in.Opening.Cash, in.Opening.Locked

	nisa, taxable, basis := in.Opening.NISA, in.Opening.Taxable, in.Opening.Basis

	maturing := in.Opening.MaturingNISA
	nisaUsed := in.Opening.NISA - in.Opening.MaturingNISA

	var borrowed money.Yen
	for _, y := range years {
		timeline, _ := in.Timeline.At(y)

		var row AssetsRow
		row.Balance = timeline.Balance

		if y <= in.StartsAfter {
			held, ok := in.Actual[y]
			if !ok {
				rows = append(rows, relation.Row[AssetsRow]{Year: y, Value: row})
				continue
			}
			row.Cash, row.Invested, row.Locked = held.Cash, held.Invested, held.Locked
			row.Total = held.Total()
			row.Recorded = true
			cash, locked = held.Cash, held.Locked
			nisa, taxable, basis = held.NISA, held.Taxable, held.Basis
			maturing = held.MaturingNISA
			nisaUsed = held.NISA - held.MaturingNISA
			row.NISA, row.Taxable, row.Basis = nisa, taxable, basis
			rows = append(rows, relation.Row[AssetsRow]{Year: y, Value: row})
			continue
		}

		cash += row.Balance

		if r := in.LastResort; r.From != 0 && r.Measure.GivesUpHome && y >= r.From {
			if y == r.From {
				row.Raised = r.Proceeds
				cash += row.Raised
			}
			row.Housing = r.Yearly(y)
			cash -= row.Housing
		}

		if in.LastResort.From != 0 && !in.LastResort.Measure.GivesUpHome && borrowed > 0 {
			row.Interest = in.LastResort.Measure.InterestOn(borrowed)
			cash -= row.Interest
		}

		if paid, ok := in.MutualAid.At(y); ok {
			row.Paid = paid
			cash -= paid
			locked += paid
		}

		if in.PensionReceivedIn != nil && y == *in.PensionReceivedIn && locked > 0 {
			row.PensionReceived = locked
			incomeTax, resident := law.RetirementIncomeTax(
				locked, in.PensionServiceYears, y, in.ResidentTax)
			row.PensionTax = incomeTax + resident

			cash += row.PensionReceived - row.PensionTax
			locked = 0
		}

		if fall, ok := in.Crash[y]; ok {
			row.Crash = (nisa + taxable).Mul(fall, money.Truncate)
			onNISA := law.GainOn(row.Crash, nisa, nisa+taxable)
			maturing += money.ShareOf(onNISA, maturing, nisa)
			nisa += onNISA
			taxable += row.Crash - onNISA
		}

		rate, ok := in.Return.At(y)
		if !ok {
			return empty, fmt.Errorf("table.AssetsTable: no return for %d", y)
		}
		row.Returns = (nisa + taxable).Mul(rate, money.Truncate)
		onNISA := law.GainOn(row.Returns, nisa, nisa+taxable)
		maturing += money.ShareOf(onNISA, maturing, nisa)
		nisa += onNISA
		taxable += row.Returns - onNISA

		written, ok := in.CashFloor.At(y)
		if !ok {
			return empty, fmt.Errorf("table.AssetsTable: no cash floor for %d", y)
		}
		floor := written
		written, ok = in.ContributionMonthly.At(y)
		if !ok {
			return empty, fmt.Errorf("table.AssetsTable: no contribution for %d", y)
		}
		monthly := written

		switch {
		case cash < 0:
			sellNISA := func() {
				fromNISA := min(-cash, nisa)
				maturing -= money.ShareOf(fromNISA, maturing, nisa)
				nisa -= fromNISA
				row.Withdrawn += fromNISA
				cash += fromNISA
			}
			sellTaxable := func() {
				sold, tax := law.SellForCash(-cash, taxable, basis)
				if sold == 0 {
					return
				}
				basis -= law.GainOn(sold, basis, taxable)
				taxable -= sold
				row.Withdrawn += sold
				row.InvestmentTax += tax
				cash += sold - tax
			}

			if in.SellNISAFirst {
				sellNISA()
				sellTaxable()
			} else {
				sellTaxable()
				sellNISA()
			}

			if r := in.LastResort; r.From != 0 && !r.Measure.GivesUpHome && y >= r.From && cash < 0 {
				draw := min(shortfallOf(cash), r.Proceeds-borrowed)
				if draw > 0 {
					borrowed += draw
					row.Raised = draw
					cash += draw
				}
			}

			if cash < 0 {
				row.Shortfall = shortfallOf(cash)
				cash = 0
			}

		case cash > floor:
			row.Contributed = min(cash-floor, monthly*date.MonthsAYear)
			cash -= row.Contributed
			allowance, ok := in.NISAAllowance.At(y)
			if !ok {
				return empty, fmt.Errorf("table.AssetsTable: the NISA 生涯投資枠 of %d is unknown", y)
			}
			toNISA := min(row.Contributed, max(allowance-nisaUsed, 0))
			nisa += toNISA
			nisaUsed += toNISA
			taxable += row.Contributed - toNISA
			basis += row.Contributed - toNISA
		}

		if maturing > 0 && y >= in.MaturityOfOldNISA && in.MaturityOfOldNISA != 0 {
			nisa -= maturing
			taxable += maturing
			basis += law.NISAMaturityBasis(maturing)
			row.MaturedFromNISA = maturing
			maturing = 0
		}

		row.Cash, row.Locked = cash, locked
		row.NISA, row.Taxable, row.Basis = nisa, taxable, basis
		row.Invested = nisa + taxable
		row.Total = cash + row.Invested + locked

		rows = append(rows, relation.Row[AssetsRow]{Year: y, Value: row})
	}

	if locked != 0 {
		return empty, fmt.Errorf(
			"table.AssetsTable: 計画の最終年に %d の年金資産が残っている。年金資産は受け取り切って 0 になるはずで、残っていれば世帯の資産が消え、負であれば無かった金を数えている。受給する年が決まっていないか、受給年より後に掛金が続いているか、掛金が負である",
			locked)
	}

	return relation.New(rows), nil
}
