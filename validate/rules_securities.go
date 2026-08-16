package validate

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	HoldingsFollowLedgerRule RuleName = "securities-holdings-follow-ledger"
	StatementsCoverRule      RuleName = "securities-statements-cover"
	HoldingsFollowPricesRule RuleName = "securities-holdings-follow-prices"
)

var QuarterEnds = [...]string{"-03-31", "-06-30", "-09-30", "-12-31"}

func StatementsCoverEveryQuarter(slot tsv.Slot, asOfColumn tsv.ColumnName, from, through string) Rule {
	return Rule{
		Name:  StatementsCoverRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, asOfColumn)
			if len(found) > 0 {
				return found
			}

			held := make(map[string]bool, len(table.Rows))
			for _, fields := range table.Rows {
				held[fields[at[asOfColumn]]] = true
			}

			want := make(map[string]bool)
			for year := from[:4]; year <= through[:4]; year = nextYear(year) {
				for _, end := range QuarterEnds {
					if asOf := year + end; from <= asOf && asOf <= through {
						want[asOf] = true
					}
				}
			}

			missing := make([]string, 0, len(want))
			for asOf := range want {
				if !held[asOf] {
					missing = append(missing, asOf)
				}
			}
			slices.Sort(missing)
			for _, asOf := range missing {
				found = append(found, Finding{
					Slot:    slot,
					Message: fmt.Sprintf("%s の報告書が無い。%s から %s まで四半期ごとに要る", asOf, from, through),
				})
			}

			spare := make([]string, 0, len(held))
			for asOf := range held {
				if !want[asOf] {
					spare = append(spare, asOf)
				}
			}
			slices.Sort(spare)
			for _, asOf := range spare {
				found = append(found, Finding{
					Slot: slot,
					Message: fmt.Sprintf("%s は %s から %s までの四半期末ではない。範囲を広げるならここに書く",
						asOf, from, through),
				})
			}
			return found
		},
	}
}

func nextYear(year string) string {
	n, err := strconv.Atoi(year)
	if err != nil {
		return year + "!"
	}
	return strconv.Itoa(n + 1)
}

type holding struct {
	fund   string
	pocket string
}

func (h holding) String() string { return h.fund + " " + h.pocket }

type LedgerVocabulary struct {
	Pockets map[string]string

	Bought, Sold string
}

func HoldingsFollowTheLedger(holdings HoldingSide, ledger LedgerSide, deals LedgerVocabulary) Rule {
	holdingsSlot, ledgerSlot := holdings.Slot, ledger.Slot
	asOfColumn, holdingFundColumn := holdings.AsOf, holdings.Fund
	pocketColumn, holdingUnitsColumn := holdings.Pocket, holdings.Units
	tradedColumn, ledgerFundColumn := ledger.Traded, ledger.Fund
	depositColumn, dealColumn, ledgerUnitsColumn := ledger.Deposit, ledger.Deal, ledger.Units

	pockets, bought, sold := deals.Pockets, deals.Bought, deals.Sold
	return Rule{
		Name:  HoldingsFollowLedgerRule,
		Needs: []tsv.Slot{holdingsSlot, ledgerSlot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			holdings := tables[holdingsSlot]
			ledger := tables[ledgerSlot]

			atHolding, found := columnsOf(holdings, holdingsSlot,
				asOfColumn, holdingFundColumn, pocketColumn, holdingUnitsColumn)
			atLedger, missing := columnsOf(ledger, ledgerSlot,
				tradedColumn, ledgerFundColumn, depositColumn, dealColumn, ledgerUnitsColumn)
			found = append(found, missing...)
			if len(found) > 0 {
				return found
			}

			named := make(map[string]bool, len(pockets))
			for _, pocket := range pockets {
				named[pocket] = true
			}

			reported := make(map[string]map[holding]int64)
			asOfs := make([]string, 0, len(holdings.Rows))
			for row, fields := range holdings.Rows {
				units, err := wholeNumber(fields[atHolding[holdingUnitsColumn]])
				if err != nil {
					found = append(found, Finding{
						Slot:    holdingsSlot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, holdingUnitsColumn, err),
					})
					continue
				}
				if pocket := fields[atHolding[pocketColumn]]; !named[pocket] {
					found = append(found, Finding{
						Slot: holdingsSlot,
						Message: fmt.Sprintf("%d 行目: %s が %q で、台帳のどの %s にも対応しない",
							row+1, pocketColumn, pocket, depositColumn),
					})
					continue
				}
				asOf := fields[atHolding[asOfColumn]]
				if _, seen := reported[asOf]; !seen {
					reported[asOf] = make(map[holding]int64)
					asOfs = append(asOfs, asOf)
				}
				reported[asOf][holding{
					fund:   fields[atHolding[holdingFundColumn]],
					pocket: fields[atHolding[pocketColumn]],
				}] += units
			}
			if len(found) > 0 {
				return found
			}
			slices.Sort(asOfs)

			type trade struct {
				traded string
				at     holding
				units  int64
			}
			trades := make([]trade, 0, len(ledger.Rows))
			for row, fields := range ledger.Rows {
				units, err := wholeNumber(fields[atLedger[ledgerUnitsColumn]])
				if err != nil {
					found = append(found, Finding{
						Slot:    ledgerSlot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, ledgerUnitsColumn, err),
					})
					continue
				}
				deposit := fields[atLedger[depositColumn]]
				pocket, known := pockets[deposit]
				if !known {
					found = append(found, Finding{
						Slot: ledgerSlot,
						Message: fmt.Sprintf("%d 行目: %s が %q で、対応する %s を知らない",
							row+1, depositColumn, deposit, pocketColumn),
					})
					continue
				}
				switch deal := fields[atLedger[dealColumn]]; deal {
				case bought:
				case sold:
					units = -units
				default:
					found = append(found, Finding{
						Slot: ledgerSlot,
						Message: fmt.Sprintf("%d 行目: %s が %q で、%q でも %q でもない",
							row+1, dealColumn, deal, bought, sold),
					})
					continue
				}
				trades = append(trades, trade{
					traded: fields[atLedger[tradedColumn]],
					at:     holding{fund: fields[atLedger[ledgerFundColumn]], pocket: pocket},
					units:  units,
				})
			}
			if len(found) > 0 {
				return found
			}
			slices.SortFunc(trades, func(a, b trade) int {
				return cmp.Compare(a.traded, b.traded)
			})

			running := make(map[holding]int64)
			next := 0
			for _, asOf := range asOfs {
				for next < len(trades) && trades[next].traded <= asOf {
					running[trades[next].at] += trades[next].units
					next++
				}

				at := make([]holding, 0, len(running)+len(reported[asOf]))
				for pot, units := range running {
					if units != 0 {
						at = append(at, pot)
					}
				}
				for pot := range reported[asOf] {
					if units, held := running[pot]; !held || units == 0 {
						at = append(at, pot)
					}
				}
				slices.SortFunc(at, func(a, b holding) int { return cmp.Compare(a.String(), b.String()) })

				for _, pot := range at {
					if want, got := running[pot], reported[asOf][pot]; want != got {
						found = append(found, Finding{
							Slot: holdingsSlot,
							Message: fmt.Sprintf("%s の %s: 報告書は %d 口、台帳を積むと %d 口",
								asOf, pot, got, want),
						})
					}
				}
			}
			return found
		},
	}
}

func centiYen(field string) (int64, error) {
	negative := strings.HasPrefix(field, "-")

	whole, frac, hasFrac := strings.Cut(field, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("数でない: %q", field)
	}
	if !hasFrac {
		return w * 100, nil
	}
	if len(frac) != 2 {
		return 0, fmt.Errorf("小数点以下が2桁でない: %q", field)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("数でない: %q", field)
	}
	if negative {
		f = -f
	}
	return w*100 + f, nil
}

func HoldingsValueFollowsThePrices(
	slot tsv.Slot,
	unitsColumn, marketPriceColumn, marketValueColumn, bookPriceColumn, gainColumn tsv.ColumnName,
) Rule {
	return Rule{
		Name:  HoldingsFollowPricesRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot,
				unitsColumn, marketPriceColumn, marketValueColumn, bookPriceColumn, gainColumn)
			if len(found) > 0 {
				return found
			}

			for row, fields := range table.Rows {
				units, err := wholeNumber(fields[at[unitsColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, unitsColumn, err)})
					continue
				}
				marketPrice, err := centiYen(fields[at[marketPriceColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, marketPriceColumn, err)})
					continue
				}
				marketValue, err := wholeNumber(fields[at[marketValueColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, marketValueColumn, err)})
					continue
				}
				bookPrice, err := centiYen(fields[at[bookPriceColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, bookPriceColumn, err)})
					continue
				}
				gain, err := wholeNumber(fields[at[gainColumn]])
				if err != nil {
					found = append(found, Finding{Slot: slot,
						Message: fmt.Sprintf("%d 行目: %s: %v", row+1, gainColumn, err)})
					continue
				}

				fromOnePrice := units / 2_000_000
				oneRoundedPrice := fromOnePrice + 1
				twoRoundedPrices := 2*fromOnePrice + 1

				wantValue := money.HalfUp(units*marketPrice, 1_000_000)
				if diff := marketValue - wantValue; diff > oneRoundedPrice || diff < -oneRoundedPrice {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"%d 行目: %s が %d 円で、%s × %s ÷ 10,000 の %d 円と %+d 円ちがう（許容 ±%d 円）",
							row+1, marketValueColumn, marketValue, unitsColumn, marketPriceColumn, wantValue, diff, oneRoundedPrice),
					})
				}

				wantGain := money.HalfUp((marketPrice-bookPrice)*units, 1_000_000)
				if diff := gain - wantGain; diff > twoRoundedPrices || diff < -twoRoundedPrices {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf(
							"%d 行目: %s が %d 円で、(%s − %s) × %s ÷ 10,000 の %d 円と %+d 円ちがう（許容 ±%d 円）",
							row+1, gainColumn, gain, marketPriceColumn, bookPriceColumn, unitsColumn, wantGain, diff, twoRoundedPrices),
					})
				}
			}
			return found
		},
	}
}

type HoldingSide struct {
	Slot tsv.Slot

	AsOf, Fund, Pocket, Units tsv.ColumnName
}

type LedgerSide struct {
	Slot tsv.Slot

	Traded, Fund, Deposit, Deal, Units tsv.ColumnName
}
