package actuals

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const HeldAccountsPath tsv.Slot = "actuals/mf-accounts.tsv"

const (
	HeldAccountColumn tsv.ColumnName = "口座"

	HeldContainsColumn tsv.ColumnName = "内容に含まれる"

	HeldFromColumn tsv.ColumnName = "適用開始年月"
	HeldToColumn   tsv.ColumnName = "適用終了年月"

	HeldItemColumn tsv.ColumnName = "入金の費目"

	HeldSourceColumn tsv.ColumnName = "出典"
)

type held struct {
	account  string
	contains string
	from, to string

	rule Rule
}

type HeldAccounts struct {
	byAccount map[string]bool
	rules     []held

	untrusted map[string]string
}

func ParseHeldAccounts(table *tsv.Table) (HeldAccounts, error) {
	var empty HeldAccounts

	r, err := tsv.NewReader(table, HeldAccountsPath,
		HeldAccountColumn, HeldContainsColumn, HeldFromColumn, HeldToColumn,
		HeldItemColumn, HeldSourceColumn)
	if err != nil {
		return empty, fmt.Errorf("actuals.ParseHeldAccounts: %w", err)
	}

	written := make(map[[2]string]bool, r.Rows())

	out := HeldAccounts{
		byAccount: make(map[string]bool, r.Rows()),
		untrusted: make(map[string]string, 8),
	}
	for row := range r.Rows() {
		account := r.Field(row, HeldAccountColumn)
		if account == "" {
			return empty, r.Errorf(row, HeldAccountColumn, "口座が空である")
		}
		key := [2]string{account, r.Field(row, HeldContainsColumn)}
		if written[key] {
			return empty, wording.DuplicateKeyError(
				fmt.Sprintf("actuals: %s: row %d, column %q", r.Slot(), row+1, HeldAccountColumn),
				string(HeldAccountColumn)+"／"+string(HeldContainsColumn),
				wording.Pair(key[0], key[1]), wording.WhichHoldingItCountsAsEn)
		}
		written[key] = true
		out.byAccount[account] = true

		if r.Field(row, HeldSourceColumn) == "" {
			return empty, r.Errorf(row, HeldSourceColumn, "出典が空である")
		}

		from, to := r.Field(row, HeldFromColumn), r.Field(row, HeldToColumn)
		for _, c := range []struct {
			column tsv.ColumnName
			month  string
		}{{HeldFromColumn, from}, {HeldToColumn, to}} {
			if c.month != "" && !isMonth(c.month) {
				return empty, r.Errorf(row, c.column, "%q は年月ではない。yyyy-mm と書くこと", c.month)
			}
		}
		if from != "" && to != "" && from > to {
			return empty, r.Errorf(row, HeldFromColumn,
				"適用開始年月 %q が適用終了年月 %q より後である", from, to)
		}

		if item := r.Field(row, HeldItemColumn); item != "" {
			if before, twice := out.untrusted[account]; twice && before != item {
				return empty, r.Errorf(row, HeldItemColumn,
					"%q の入金の費目が %q とも %q とも書かれている", account, before, item)
			}
			out.untrusted[account] = item
		}

		if contains := r.Field(row, HeldContainsColumn); contains != "" {
			out.rules = append(out.rules, held{
				account: account, contains: foldForMatch(contains), from: from, to: to,
				rule: Rule{Table: HeldAccountsPath, Row: row + 1},
			})
		}
	}
	return out, nil
}

func (c ImportRules) WithHeldAccounts(accounts *tsv.Table) (ImportRules, error) {
	held, err := ParseHeldAccounts(accounts)
	if err != nil {
		return c, err
	}
	c.held = held
	return c, nil
}

func (h HeldAccounts) BalanceUntrusted(account string) (string, bool) {
	item, ok := h.untrusted[account]
	return item, ok
}

func (h HeldAccounts) Accounts() []string {
	out := make([]string, 0, len(h.byAccount))
	for account := range h.byAccount {
		out = append(out, account)
	}
	slices.Sort(out)
	return out
}

func (h HeldAccounts) Holds(account string) bool { return h.byAccount[account] }

type MarkUsed func(rule Rule)

func (u Used) Into() MarkUsed { return func(rule Rule) { u[rule] = true } }

func DiscardUsed(Rule) {}

func (h held) name() string {
	s := h.account + "/" + h.contains
	if h.from != "" || h.to != "" {
		s += "[" + h.from + "〜" + h.to + "]"
	}
	return s
}

func (h held) matches(content, month string) bool {
	return strings.Contains(content, h.contains) &&
		(h.from == "" || month >= h.from) &&
		(h.to == "" || month <= h.to)
}

func (h HeldAccounts) IsMoveInto(account, content, month string, mark MarkUsed) bool {
	if h.byAccount[account] {
		return false
	}
	folded := foldForMatch(content)
	for _, rule := range h.rules {
		if rule.matches(folded, month) {
			mark(rule.rule)
			return true
		}
	}
	return false
}

func foldForMatch(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= '！' && r <= '～':
			b.WriteRune(r - 0xFEE0)
		case r == '　' || r == ' ':
		default:
			b.WriteRune(r)
		}
	}
	return strings.ToUpper(b.String())
}
