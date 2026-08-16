package actuals

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	MoneyForwardCountedColumn  tsv.ColumnName = "計算対象"
	MoneyForwardDateColumn     tsv.ColumnName = "日付"
	MoneyForwardAmountColumn   tsv.ColumnName = "金額（円）"
	MoneyForwardMajorColumn    tsv.ColumnName = "大項目"
	MoneyForwardMinorColumn    tsv.ColumnName = "中項目"
	MoneyForwardContentColumn  tsv.ColumnName = "内容"
	MoneyForwardAccountColumn  tsv.ColumnName = "保有金融機関"
	MoneyForwardTransferColumn tsv.ColumnName = "振替"
)

const (
	PayeeAccountColumn  tsv.ColumnName = "保有金融機関"
	PayeeMajorColumn    tsv.ColumnName = "大項目"
	PayeeContainsColumn tsv.ColumnName = "内容に含まれる"
	PayeeItemColumn     tsv.ColumnName = "費目"

	PayeeVerdictColumn tsv.ColumnName = "扱い"

	PayeeFromColumn tsv.ColumnName = "適用開始年月"
	PayeeToColumn   tsv.ColumnName = "適用終了年月"
)

const (
	UnseenCard Item = "カード（明細なし）"

	UnseenCash Item = "現金支出（明細なし）"
)

const (
	ExcludedMajorColumn   tsv.ColumnName = "大項目"
	ExcludedMinorColumn   tsv.ColumnName = "中項目"
	ExcludedVerdictColumn tsv.ColumnName = "扱い"
	ExcludedItemColumn    tsv.ColumnName = "費目"

	ExcludedAccountColumn tsv.ColumnName = "保有金融機関"

	ExcludedMarkColumn tsv.ColumnName = "MF振替印"

	ExcludedFromColumn tsv.ColumnName = "適用開始年月"
	ExcludedToColumn   tsv.ColumnName = "適用終了年月"
)

type Verdict string

const (
	ExcludedCounted Verdict = "数える"

	ExcludedTransfer Verdict = "振替"

	ExcludedElsewhere Verdict = "計画が別に持つ"

	ExcludedUnknown Verdict = "不明"
)

var ExcludedVerdicts = []Verdict{
	ExcludedCounted, ExcludedTransfer, ExcludedElsewhere, ExcludedUnknown,
}

const (
	CategoryMajorColumn tsv.ColumnName = "大項目"
	CategoryMinorColumn tsv.ColumnName = "中項目"
	CategoryItemColumn  tsv.ColumnName = "費目"

	CashflowMonthColumn  tsv.ColumnName = "年月"
	CashflowItemColumn   tsv.ColumnName = "費目"
	CashflowAmountColumn tsv.ColumnName = "金額[円]"
)

type Answer string

const (
	Yes Answer = "はい"
	No  Answer = "いいえ"
)

type Item string

const Unclassified Item = "未分類"

func (i Item) Known() bool {
	return slices.Contains(KnownItems, i)
}

type Treatment struct {
	verdict Verdict
	item    Item
}

func counted(item Item) Treatment { return Treatment{verdict: ExcludedCounted, item: item} }

func dropped(verdict Verdict) Treatment { return Treatment{verdict: verdict} }

func (t Treatment) Count() (Item, bool) {
	return t.item, t.verdict == ExcludedCounted
}

func (t Treatment) Verdict() Verdict { return t.verdict }

func ParseTreatment(verdict Verdict, item Item, allowed []Verdict) (Treatment, error) {
	if !slices.Contains(allowed, verdict) {
		words := make([]string, 0, len(allowed))
		for _, v := range allowed {
			words = append(words, string(v))
		}
		return Treatment{}, fmt.Errorf(
			"扱いが %q である。%s のいずれかであること", verdict, strings.Join(words, "・"))
	}
	if verdict != ExcludedCounted {
		if item != "" {
			return Treatment{}, fmt.Errorf("扱いが %q なのに費目 %q が書いてある", verdict, item)
		}
		return dropped(verdict), nil
	}
	if item == "" {
		return Treatment{}, fmt.Errorf("扱いが %q なのに費目が空である", ExcludedCounted)
	}
	if !item.Known() {
		return Treatment{}, fmt.Errorf(
			"費目 %q は計画が持たない。actuals.KnownItems のいずれかを書くこと", item)
	}
	return counted(item), nil
}

var PayeeVerdicts = []Verdict{ExcludedCounted, ExcludedTransfer, ExcludedElsewhere}

type ImportRules struct {
	byPair     map[[2]string]Item
	byPayee    []payee
	byExcluded []excluded
	held       HeldAccounts
}

type excluded struct {
	rule Rule

	major, minor string

	account string

	mark Answer

	from, to string

	treatment Treatment
}

func (e excluded) matches(major, minor, account, month string, marked bool) bool {
	return e.major == major && e.minor == minor &&
		(e.account == "" || e.account == account) &&
		(e.mark == "" || (e.mark == Yes) == marked) &&
		(e.from == "" || month >= e.from) &&
		(e.to == "" || month <= e.to)
}

func (e excluded) covers(f excluded) bool {
	return e.major == f.major && e.minor == f.minor &&
		(e.account == "" || e.account == f.account) &&
		(e.mark == "" || e.mark == f.mark) &&
		(e.from == "" || (f.from != "" && f.from >= e.from)) &&
		(e.to == "" || (f.to != "" && f.to <= e.to))
}

const (
	CategoriesPath tsv.Slot = "actuals/categories.tsv"
	PayeesPath     tsv.Slot = "actuals/payees.tsv"
	ExcludedPath   tsv.Slot = "actuals/excluded.tsv"
)

type Rule struct {
	Table tsv.Slot

	Row int
}

type Used map[Rule]bool

func (e excluded) name() string {
	s := e.major + "/" + e.minor
	if e.account != "" {
		s += "（" + e.account + "）"
	}
	if e.mark != "" {
		s += "[" + string(ExcludedMarkColumn) + "=" + string(e.mark) + "]"
	}
	if e.from != "" || e.to != "" {
		s += "[" + e.from + "〜" + e.to + "]"
	}
	return s
}

func (p payee) name() string {
	s := p.account + "/" + p.major + "/" + p.contains
	if p.from != "" || p.to != "" {
		s += "[" + p.from + "〜" + p.to + "]"
	}
	return s
}

func (c ImportRules) Unused(used Used) []string {
	var missing []string
	for _, p := range c.byPayee {
		if !used[p.rule] {
			missing = append(missing, string(PayeeItemColumn)+" "+p.name())
		}
	}
	for _, e := range c.byExcluded {
		if !used[e.rule] {
			missing = append(missing, string(ExcludedVerdictColumn)+" "+e.name())
		}
	}
	for _, h := range c.held.rules {
		if !used[h.rule] {
			missing = append(missing, string(HeldContainsColumn)+" "+h.name())
		}
	}
	return missing
}

type payee struct {
	account   string
	major     string
	contains  string
	from, to  string
	treatment Treatment

	rule Rule
}

func (p payee) covers(q payee) bool {
	return (p.account == "" || p.account == q.account) &&
		(p.major == "" || p.major == q.major) &&
		(p.contains == "" || strings.Contains(foldForMatch(q.contains), foldForMatch(p.contains))) &&
		(p.from == "" || (q.from != "" && q.from >= p.from)) &&
		(p.to == "" || (q.to != "" && q.to <= p.to))
}

func (p payee) matches(account, major, content, month string) bool {
	return (p.account == "" || p.account == account) &&
		(p.major == "" || p.major == major) &&
		(p.contains == "" || strings.Contains(foldForMatch(content), foldForMatch(p.contains))) &&
		(p.from == "" || month >= p.from) &&
		(p.to == "" || month <= p.to)
}

func (c ImportRules) WithPayees(table *tsv.Table) (ImportRules, error) {
	r, err := tsv.NewReader(table, PayeesPath,
		PayeeAccountColumn, PayeeMajorColumn, PayeeContainsColumn, PayeeItemColumn,
		PayeeVerdictColumn, PayeeFromColumn, PayeeToColumn)
	if err != nil {
		return c, fmt.Errorf("actuals.WithPayees: %w", err)
	}

	payees := make([]payee, 0, r.Rows())
	for row := range r.Rows() {
		account := r.Field(row, PayeeAccountColumn)
		major := r.Field(row, PayeeMajorColumn)
		contains := r.Field(row, PayeeContainsColumn)
		if account == "" && major == "" && contains == "" {
			return c, fmt.Errorf("actuals.WithPayees: row %d narrows nothing, so it would match every transaction", row+1)
		}

		from, to := r.Field(row, PayeeFromColumn), r.Field(row, PayeeToColumn)
		for column, month := range map[tsv.ColumnName]string{PayeeFromColumn: from, PayeeToColumn: to} {
			if month == "" {
				continue
			}
			if !isMonth(month) {
				return c, fmt.Errorf(
					"actuals.WithPayees: row %d: %s が %q である。yyyy-mm と書くこと",
					row+1, column, month)
			}
		}
		if from != "" && to != "" && from > to {
			return c, fmt.Errorf(
				"actuals.WithPayees: row %d: 適用開始 %s が適用終了 %s より後である", row+1, from, to)
		}

		treatment, err := ParseTreatment(
			Verdict(r.Field(row, PayeeVerdictColumn)), Item(r.Field(row, PayeeItemColumn)), PayeeVerdicts)
		if err != nil {
			return c, fmt.Errorf("actuals.WithPayees: row %d: %w", row+1, err)
		}

		payees = append(payees, payee{
			rule:      Rule{Table: PayeesPath, Row: row + 1},
			account:   account,
			major:     major,
			contains:  contains,
			from:      from,
			to:        to,
			treatment: treatment,
		})
	}

	for j, narrow := range payees {
		for i, wide := range payees[:j] {
			if wide.covers(narrow) {
				return c, fmt.Errorf(
					"actuals.WithPayees: row %d は row %d に必ず飲み込まれるので一度も当たらない。狭い行を先に置くこと",
					j+1, i+1)
			}
		}
	}

	c.byPayee = payees
	return c, nil
}

func isMonth(s string) bool {
	if len(s) != 7 || s[4] != '-' {
		return false
	}
	for i, r := range s {
		if i == 4 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return s[5:] >= "01" && s[5:] <= "12"
}

func ImportRulesFromCategories(table *tsv.Table) (ImportRules, error) {
	r, err := tsv.NewReader(table, CategoriesPath,
		CategoryMajorColumn, CategoryMinorColumn, CategoryItemColumn)
	if err != nil {
		return ImportRules{}, fmt.Errorf("actuals.ImportRulesFromCategories: %w", err)
	}

	byPair := make(map[[2]string]Item, r.Rows())
	for row := range r.Rows() {
		pair := [2]string{r.Field(row, CategoryMajorColumn), r.Field(row, CategoryMinorColumn)}
		item := Item(r.Field(row, CategoryItemColumn))
		if item == "" {
			return ImportRules{}, fmt.Errorf(
				"actuals.ImportRulesFromCategories: row %d: %s / %s has no item; write %q rather than leave it blank",
				row+1, pair[0], pair[1], Unclassified)
		}
		if !item.Known() {
			return ImportRules{}, fmt.Errorf(
				"actuals.ImportRulesFromCategories: row %d: %s / %s の費目 %q は計画が持たない。actuals.KnownItems のいずれかを書くこと",
				row+1, pair[0], pair[1], item)
		}
		if _, duplicate := byPair[pair]; duplicate {
			return ImportRules{}, fmt.Errorf("actuals.ImportRulesFromCategories: row %d: %s / %s appears twice", row+1, pair[0], pair[1])
		}
		byPair[pair] = item
	}

	if len(byPair) == 0 {
		return ImportRules{}, fmt.Errorf("actuals.ImportRulesFromCategories: nothing is mapped, so every transaction would be dropped")
	}
	return ImportRules{byPair: byPair}, nil
}

func (c ImportRules) OfPayee(account, major, content, month string, used Used) (Treatment, bool) {
	for _, p := range c.byPayee {
		if p.matches(account, major, content, month) {
			used[p.rule] = true
			return p.treatment, true
		}
	}
	return Treatment{}, false
}

func (c ImportRules) WithExcluded(table *tsv.Table) (ImportRules, error) {
	r, err := tsv.NewReader(table, ExcludedPath,
		ExcludedMajorColumn, ExcludedMinorColumn, ExcludedAccountColumn, ExcludedMarkColumn,
		ExcludedFromColumn, ExcludedToColumn, ExcludedVerdictColumn, ExcludedItemColumn)
	if err != nil {
		return c, fmt.Errorf("actuals.WithExcluded: %w", err)
	}

	rules := make([]excluded, 0, r.Rows())
	for row := range r.Rows() {
		e := excluded{
			rule:    Rule{Table: ExcludedPath, Row: row + 1},
			major:   r.Field(row, ExcludedMajorColumn),
			minor:   r.Field(row, ExcludedMinorColumn),
			account: r.Field(row, ExcludedAccountColumn),
			mark:    Answer(r.Field(row, ExcludedMarkColumn)),
			from:    r.Field(row, ExcludedFromColumn),
			to:      r.Field(row, ExcludedToColumn),
		}

		for column, month := range map[tsv.ColumnName]string{ExcludedFromColumn: e.from, ExcludedToColumn: e.to} {
			if month != "" && !isMonth(month) {
				return c, fmt.Errorf(
					"actuals.WithExcluded: row %d: %s が %q である。yyyy-mm と書くこと", row+1, column, month)
			}
		}
		if e.from != "" && e.to != "" && e.from > e.to {
			return c, fmt.Errorf(
				"actuals.WithExcluded: row %d: 適用開始 %s が適用終了 %s より後である", row+1, e.from, e.to)
		}

		switch e.mark {
		case "", Yes, No:
		default:
			return c, fmt.Errorf(
				"actuals.WithExcluded: row %d: %s が %q である。%q・%q・空 のいずれかであること",
				row+1, ExcludedMarkColumn, e.mark, Yes, No)
		}

		treatment, err := ParseTreatment(
			Verdict(r.Field(row, ExcludedVerdictColumn)),
			Item(r.Field(row, ExcludedItemColumn)),
			ExcludedVerdicts)
		if err != nil {
			return c, fmt.Errorf(
				"actuals.WithExcluded: row %d: %q / %q: %w", row+1, e.major, e.minor, err)
		}
		e.treatment = treatment
		rules = append(rules, e)
	}

	for j, narrow := range rules {
		for i, wide := range rules[:j] {
			if wide.covers(narrow) {
				return c, fmt.Errorf(
					"actuals.WithExcluded: row %d は row %d に必ず飲み込まれるので一度も当たらない。狭い行を先に置くこと",
					j+1, i+1)
			}
		}
	}

	c.byExcluded = rules
	return c, nil
}

func (c ImportRules) OfExcluded(major, minor, account, month string, marked bool, used Used) (item Item, count bool, err error) {
	for _, e := range c.byExcluded {
		if !e.matches(major, minor, account, month, marked) {
			continue
		}
		used[e.rule] = true
		item, count := e.treatment.Count()
		if !count {
			return "", false, nil
		}
		if marked && e.mark != Yes {
			return "", false, fmt.Errorf(
				"actuals: %q / %q（%s）は %q だが、MoneyForward が振替の印を付けている。%s を %q にした行を書くか、扱いを変えること",
				major, minor, account, ExcludedCounted, ExcludedMarkColumn, Yes)
		}
		return item, true, nil
	}
	return "", false, fmt.Errorf(
		"actuals: %q / %q（%s）は計算対象外になっているが、扱いが決まっていない。黙って落とさずに excluded.tsv へ書くこと",
		major, minor, account)
}

func (c ImportRules) planHoldsElsewhere(record exportRecord, used Used) bool {
	if treatment, named := c.OfPayee(
		record.account, record.major, record.content, record.month, used,
	); named {
		return treatment.Verdict() == ExcludedElsewhere
	}
	if record.counted {
		return false
	}
	return c.excludedHoldsElsewhere(
		record.major, record.minor, record.account, record.month, record.marked, used)
}

func (c ImportRules) excludedHoldsElsewhere(major, minor, account, month string, marked bool, used Used) bool {
	for _, e := range c.byExcluded {
		if e.matches(major, minor, account, month, marked) {
			if e.treatment.Verdict() == ExcludedElsewhere {
				used[e.rule] = true
				return true
			}
			return false
		}
	}
	return false
}

func (c ImportRules) Of(major, minor string) (Item, error) {
	item, ok := c.byPair[[2]string{major, minor}]
	if !ok {
		return "", fmt.Errorf(
			"actuals: %q / %q is not mapped to an item; add it to categories.tsv rather than let it be dropped",
			major, minor)
	}
	return item, nil
}

type exportRecord struct {
	counted, marked  bool
	month            string
	amount           money.Yen
	major, minor     string
	account, content string
}

func CountExportRecords(r io.Reader) (int, error) {
	records, err := readExportRecords(r)
	if err != nil {
		return 0, fmt.Errorf("actuals.CountExportRecords: %w", err)
	}
	return len(records), nil
}

func readExportRecords(r io.Reader) ([]exportRecord, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := assertUTF8(content); err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewReader(tsv.StripByteOrderMark(content)))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no header row")
	}

	header := records[0]
	at := make(map[tsv.ColumnName]int, 8)
	for _, column := range []tsv.ColumnName{
		MoneyForwardCountedColumn, MoneyForwardDateColumn, MoneyForwardAmountColumn,
		MoneyForwardMajorColumn, MoneyForwardMinorColumn, MoneyForwardTransferColumn,
		MoneyForwardContentColumn, MoneyForwardAccountColumn,
	} {
		i := slices.Index(header, string(column))
		if i < 0 {
			return nil, fmt.Errorf("no %q column", column)
		}
		at[column] = i
	}

	out := make([]exportRecord, 0, len(records)-1)
	for i, record := range records[1:] {
		if len(record) <= at[MoneyForwardTransferColumn] {
			return nil, fmt.Errorf("row %d has %d fields, too few", i+1, len(record))
		}
		amount, err := money.ParseYen(record[at[MoneyForwardAmountColumn]])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		month, err := monthOf(record[at[MoneyForwardDateColumn]])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out = append(out, exportRecord{
			counted: record[at[MoneyForwardCountedColumn]] == "1",
			marked:  record[at[MoneyForwardTransferColumn]] == "1",
			month:   month,
			amount:  amount,
			major:   record[at[MoneyForwardMajorColumn]],
			minor:   record[at[MoneyForwardMinorColumn]],
			account: record[at[MoneyForwardAccountColumn]],
			content: record[at[MoneyForwardContentColumn]],
		})
	}
	return out, nil
}

func ImportMoneyForwardCashflow(r io.Reader, categories ImportRules) (*tsv.Table, Used, error) {
	records, err := readExportRecords(r)
	if err != nil {
		return nil, nil, fmt.Errorf("actuals.ImportMoneyForwardCashflow: %w", err)
	}

	used := make(Used, 8)

	byKey := make(map[monthItem]money.Yen)
	var transfers int

	for i, record := range records {
		if record.amount == 0 {
			continue
		}

		if categories.held.IsMoveInto(record.account, record.content, record.month, used.Into()) {
			continue
		}

		if _, untrusted := categories.held.BalanceUntrusted(record.account); untrusted {
			continue
		}

		if !record.counted {
			item, count, err := categories.OfExcluded(
				record.major, record.minor, record.account, record.month, record.marked, used)
			if err != nil {
				return nil, nil, fmt.Errorf("actuals.ImportMoneyForwardCashflow: row %d: %w", i+1, err)
			}
			if !count {
				continue
			}
			byKey[monthItem{record.month, item}] += record.amount
			continue
		}
		if record.marked {
			transfers++
			continue
		}

		var item Item
		if treatment, named := categories.OfPayee(
			record.account, record.major, record.content, record.month, used,
		); named {
			var count bool
			if item, count = treatment.Count(); !count {
				continue
			}
		} else {
			var err error
			item, err = categories.Of(record.major, record.minor)
			if err != nil {
				return nil, nil, fmt.Errorf("actuals.ImportMoneyForwardCashflow: row %d: %w", i+1, err)
			}
		}

		byKey[monthItem{record.month, item}] += record.amount
	}

	if transfers > 0 {
		return nil, nil, fmt.Errorf(
			"actuals.ImportMoneyForwardCashflow: %d transfer(s) are marked 計算対象 = 1; keeping them would count the same money twice, so decide what they are before importing",
			transfers)
	}
	return cashflowTable(byKey), used, nil
}

type monthItem struct {
	month string
	item  Item
}

func cashflowTable(byKey map[monthItem]money.Yen) *tsv.Table {
	keys := make([]monthItem, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b monthItem) int {
		if a.month != b.month {
			return strings.Compare(a.month, b.month)
		}
		return strings.Compare(string(a.item), string(b.item))
	})

	out := &tsv.Table{Header: []tsv.ColumnName{
		CashflowMonthColumn, CashflowItemColumn, CashflowAmountColumn,
	}}
	for _, k := range keys {
		out.Rows = append(out.Rows, []string{k.month, string(k.item), fmt.Sprint(int64(byKey[k]))})
	}
	return out
}

func monthOf(date string) (string, error) {
	parts := strings.Split(date, "/")
	if len(parts) != 3 || len(parts[0]) != 4 {
		return "", fmt.Errorf("%q is not a date of the form 2025/12/31", date)
	}
	return fmt.Sprintf("%s-%02s", parts[0], parts[1]), nil
}

func MergeCashflow(tables []*tsv.Table) (*tsv.Table, error) {
	byKey := make(map[monthItem]money.Yen)

	for _, table := range tables {
		r, err := tsv.NewReader(table, CashflowPath,
			CashflowMonthColumn, CashflowItemColumn, CashflowAmountColumn)
		if err != nil {
			return nil, fmt.Errorf("actuals.MergeCashflow: %w", err)
		}
		for row := range r.Rows() {
			amount, err := money.ParseYen(r.Field(row, CashflowAmountColumn))
			if err != nil {
				return nil, fmt.Errorf("actuals.MergeCashflow: %w",
					r.Errorf(row, CashflowAmountColumn, "%v", err))
			}
			key := monthItem{r.Field(row, CashflowMonthColumn), Item(r.Field(row, CashflowItemColumn))}
			byKey[key] += amount
		}
	}

	return cashflowTable(byKey), nil
}
