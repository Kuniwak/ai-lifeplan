package actuals

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

func (c ImportRules) spends(records []exportRecord, used Used) (*tsv.Table, error) {
	out := &tsv.Table{Header: []tsv.ColumnName{
		CashflowMonthColumn, CashflowItemColumn, CashflowAmountColumn,
	}}
	inflows := c.inflows(records, used)

	for _, record := range records {
		item, untrusted := c.held.BalanceUntrusted(record.account)
		if !untrusted || record.amount == 0 {
			continue
		}

		planHolds := c.planHoldsElsewhere(record, used)
		switch {
		case record.amount > 0 && planHolds:
			continue
		case record.amount > 0:
		case record.amount < 0 && planHolds:
			booked, err := inflows.take(record)
			if err != nil {
				return nil, err
			}
			if !booked {
				continue
			}
		default:
			continue
		}

		out.Rows = append(out.Rows, []string{
			record.month, item, fmt.Sprint(int64(-record.amount)),
		})
	}
	return out, nil
}

type inflowKey struct {
	account, month string
	amount         money.Yen
}

type inflowsSeen map[inflowKey]*struct{ booked, held int }

func (c ImportRules) inflows(records []exportRecord, used Used) inflowsSeen {
	seen := make(inflowsSeen, 32)
	for _, record := range records {
		if _, untrusted := c.held.BalanceUntrusted(record.account); !untrusted {
			continue
		}
		if record.amount <= 0 {
			continue
		}
		key := inflowKey{record.account, record.month, record.amount}
		count := seen[key]
		if count == nil {
			count = &struct{ booked, held int }{}
			seen[key] = count
		}
		if c.planHoldsElsewhere(record, used) {
			count.held++
		} else {
			count.booked++
		}
	}
	return seen
}

func (s inflowsSeen) take(record exportRecord) (booked bool, err error) {
	key := inflowKey{record.account, record.month, -record.amount}
	count := s[key]
	switch {
	case count == nil || count.booked+count.held == 0:
		return false, fmt.Errorf(
			"actuals: %s の %s から %s 円が「計画が別に持つ」として出ていくが、"+
				"同じ額がその月にこの口座へ入っていない。"+
				"入った月と出ていった月がずれているか、入った側が別の額で記録されている。"+
				"**入ってきた行は自分が何の金かを名乗らないので、組にできるのは額だけである**——"+
				"MoneyForward 側で入金の行にも印を付けるか、月をまたがないように記録すること",
			record.month, record.account, withSeparators(int64(-record.amount)))
	case count.held > 0:
		count.held--
		return false, nil
	default:
		count.booked--
		return true, nil
	}
}

const UntrustedPath tsv.Slot = "actuals/untrusted.tsv"

const (
	UntrustedAccountColumn tsv.ColumnName = "口座"
	UntrustedInColumn      tsv.ColumnName = "入金[円]"
	UntrustedOutColumn     tsv.ColumnName = "出金[円]"
	UntrustedGapColumn     tsv.ColumnName = "差[円]"
)

func (c ImportRules) gaps(records []exportRecord) *tsv.Table {
	type flow struct{ in, out money.Yen }

	byAccount := make(map[string]flow, 8)
	for _, record := range records {
		if _, untrusted := c.held.BalanceUntrusted(record.account); !untrusted {
			continue
		}
		f := byAccount[record.account]
		if record.amount > 0 {
			f.in += record.amount
		} else {
			f.out += record.amount
		}
		byAccount[record.account] = f
	}

	accounts := make([]string, 0, len(byAccount))
	for account := range byAccount {
		accounts = append(accounts, account)
	}
	slices.Sort(accounts)

	out := &tsv.Table{Header: []tsv.ColumnName{
		UntrustedAccountColumn, UntrustedInColumn, UntrustedOutColumn, UntrustedGapColumn,
	}}
	for _, account := range accounts {
		f := byAccount[account]
		out.Rows = append(out.Rows, []string{
			account,
			fmt.Sprint(int64(f.in)), fmt.Sprint(int64(f.out)), fmt.Sprint(int64(f.in + f.out)),
		})
	}
	return out
}
