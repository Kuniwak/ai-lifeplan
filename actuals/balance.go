package actuals

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const (
	BalanceYearColumn     tsv.ColumnName = "西暦"
	BalanceCashColumn     tsv.ColumnName = "貯蓄[円]"
	BalanceInvestedColumn tsv.ColumnName = "金融資産[円]"

	BalanceNISAColumn tsv.ColumnName = "金融資産(NISA)[円]"

	BalanceMaturingNISAColumn tsv.ColumnName = "金融資産(NISA)のうち非課税期間が終わるもの[円]"

	BalanceTaxableColumn tsv.ColumnName = "金融資産(課税)[円]"
	BalanceBasisColumn   tsv.ColumnName = "金融資産(課税)の取得価額[円]"
	BalanceLockedColumn  tsv.ColumnName = "年金資産[円]"
	BalanceTotalColumn   tsv.ColumnName = "総資産[円]"

	BalancePartialColumn tsv.ColumnName = "一部未記録"
)

type PartialFlag string

const PartialYes PartialFlag = "はい"

const (
	CashflowPath tsv.Slot = "actuals/cashflow.tsv"
	BalancePath  tsv.Slot = "actuals/balance.tsv"

	BankBalancePath tsv.Slot = "actuals/bank-balance.tsv"
	KnownPath       tsv.Slot = "actuals/balance-known.tsv"

	BankAccountsPath tsv.Slot = "actuals/bank-accounts.tsv"

	AccountsPath tsv.Slot = "actuals/accounts.tsv"
)

const (
	KnownYearColumn    tsv.ColumnName = "西暦"
	KnownKindColumn    tsv.ColumnName = "資産種別"
	KnownBalanceColumn tsv.ColumnName = "残高[円]"
)

type Known map[date.Year]map[string]money.Yen

func ParseKnown(table *tsv.Table) (Known, error) {
	r, err := tsv.NewReader(table, KnownPath, KnownYearColumn, KnownKindColumn, KnownBalanceColumn)
	if err != nil {
		return nil, fmt.Errorf("actuals.ParseKnown: %w", err)
	}

	known := make(Known, r.Rows())
	for row := range r.Rows() {
		year, err := date.ParseYear(r.Field(row, KnownYearColumn))
		if err != nil {
			return nil, r.Errorf(row, KnownYearColumn, "%v", err)
		}
		balance, err := money.ParseYen(r.Field(row, KnownBalanceColumn))
		if err != nil {
			return nil, r.Errorf(row, KnownBalanceColumn, "%v", err)
		}

		kind := r.Field(row, KnownKindColumn)
		if known[year] == nil {
			known[year] = make(map[string]money.Yen, 1)
		}
		if _, twice := known[year][kind]; twice {
			return nil, wording.DuplicateKeyError(
				fmt.Sprintf("actuals: %s: row %d, column %q", r.Slot(), row+1, KnownKindColumn),
				fmt.Sprintf("%d %s", year, KnownKindColumn), wording.Name(kind),
				wording.WhichAmountIsTheRecordEn)
		}
		known[year][kind] = balance
	}
	return known, nil
}

const (
	AccountKindColumn   tsv.ColumnName = "資産種別"
	AccountBucketColumn tsv.ColumnName = "区分"
)

type AssetKind string

const (
	Cash AssetKind = "貯蓄"

	Invested AssetKind = "金融資産"

	Locked AssetKind = "年金資産"
)

const AccountKnownFromColumn tsv.ColumnName = "記録開始"

type Balance struct {
	Cash, Invested, Locked money.Yen

	NISA, Taxable, Basis money.Yen

	MaturingNISA money.Yen

	Partial bool
}

func (b Balance) Total() money.Yen { return b.Cash + b.Invested + b.Locked }

func (b Balance) Available() money.Yen { return b.Cash + b.Invested }

type Buckets struct {
	byKind    map[string]AssetKind
	knownFrom map[string]string
}

func ParseBuckets(table *tsv.Table) (Buckets, error) {
	kindAt, ok := table.ColumnIndex(AccountKindColumn)
	if !ok {
		return Buckets{}, fmt.Errorf("actuals.ParseBuckets: no %q column", AccountKindColumn)
	}
	bucketAt, ok := table.ColumnIndex(AccountBucketColumn)
	if !ok {
		return Buckets{}, fmt.Errorf("actuals.ParseBuckets: no %q column", AccountBucketColumn)
	}

	knownAt, hasKnown := table.ColumnIndex(AccountKnownFromColumn)

	byKind := make(map[string]AssetKind, len(table.Rows))
	knownFrom := make(map[string]string, len(table.Rows))
	for row, fields := range table.Rows {
		kind, bucket := fields[kindAt], AssetKind(fields[bucketAt])
		if bucket != Cash && bucket != Invested && bucket != Locked {
			return Buckets{}, fmt.Errorf(
				"actuals.ParseBuckets: row %d: %q goes in %q, which is none of %q, %q or %q",
				row+1, kind, bucket, Cash, Invested, Locked)
		}
		if _, duplicate := byKind[kind]; duplicate {
			return Buckets{}, fmt.Errorf("actuals.ParseBuckets: row %d: %q is assigned twice", row+1, kind)
		}
		byKind[kind] = bucket
		if hasKnown {
			from := fields[knownAt]
			if !isExportDate(from) {
				return Buckets{}, fmt.Errorf(
					"actuals.ParseBuckets: row %d: %q の %q は %q の形で書く。書き出しの日付と文字どおり比べるので、形が違えば比較が静かに逆を答える",
					row+1, kind, from, "2026/08/03")
			}
			knownFrom[kind] = from
		}
	}

	if len(byKind) == 0 {
		return Buckets{}, fmt.Errorf("actuals.ParseBuckets: nothing is assigned, so every asset would be dropped")
	}
	return Buckets{byKind: byKind, knownFrom: knownFrom}, nil
}

func isExportDate(date string) bool {
	if date == "" {
		return true
	}
	if len(date) != len("2026/08/03") {
		return false
	}
	for i, r := range date {
		digit := i != 4 && i != 7
		if digit != (r >= '0' && r <= '9') || (!digit && r != '/') {
			return false
		}
	}
	return true
}

func (b Buckets) KnownAt(kind, date string) bool {
	from, ok := b.knownFrom[kind]
	return !ok || from == "" || date >= from
}

func (b Buckets) Of(kind string) (AssetKind, error) {
	bucket, ok := b.byKind[kind]
	if !ok {
		return "", fmt.Errorf(
			"actuals: %q is not assigned to %q or %q; add it to accounts.tsv rather than let it be dropped",
			kind, Cash, Invested)
	}
	return bucket, nil
}

func (b Buckets) Kinds() []string {
	return slices.Sorted(maps.Keys(b.byKind))
}

type BalanceTable struct {
	byYear relation.Table[Balance]
}

func ParseBalanceTable(table *tsv.Table) (BalanceTable, error) {
	index := make(map[tsv.ColumnName]int, 4)
	for _, column := range []tsv.ColumnName{BalanceYearColumn, BalanceCashColumn, BalanceInvestedColumn} {
		i, ok := table.ColumnIndex(column)
		if !ok {
			return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: no %q column", column)
		}
		index[column] = i
	}

	rows := make([]relation.Row[Balance], 0, len(table.Rows))
	for row, fields := range table.Rows {
		year, err := date.ParseYear(fields[index[BalanceYearColumn]])
		if err != nil {
			return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: row %d: %w", row+1, err)
		}
		cash, err := money.ParseYen(fields[index[BalanceCashColumn]])
		if err != nil {
			return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: row %d: %w", row+1, err)
		}
		invested, err := money.ParseYen(fields[index[BalanceInvestedColumn]])
		if err != nil {
			return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: row %d: %w", row+1, err)
		}
		value := Balance{Cash: cash, Invested: invested}
		for _, c := range []struct {
			column tsv.ColumnName
			into   *money.Yen
		}{
			{BalanceNISAColumn, &value.NISA},
			{BalanceMaturingNISAColumn, &value.MaturingNISA},
			{BalanceTaxableColumn, &value.Taxable},
			{BalanceBasisColumn, &value.Basis},
		} {
			i, ok := table.ColumnIndex(c.column)
			if !ok {
				continue
			}
			if *c.into, err = money.ParseYen(fields[i]); err != nil {
				return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: row %d: %w", row+1, err)
			}
		}

		if value.NISA+value.Taxable != value.Invested {
			return BalanceTable{}, fmt.Errorf(
				"actuals.ParseBalanceTable: row %d: NISA %d ＋ 課税 %d が金融資産 %d と合わない",
				row+1, value.NISA, value.Taxable, value.Invested)
		}
		if value.MaturingNISA > value.NISA {
			return BalanceTable{}, fmt.Errorf(
				"actuals.ParseBalanceTable: row %d: 非課税期間が終わるもの %d が NISA %d を上回っている",
				row+1, value.MaturingNISA, value.NISA)
		}
		if value.MaturingNISA < 0 {
			return BalanceTable{}, fmt.Errorf(
				"actuals.ParseBalanceTable: row %d: 非課税期間が終わるもの %d が負である",
				row+1, value.MaturingNISA)
		}
		if value.Basis > value.Taxable {
			return BalanceTable{}, fmt.Errorf(
				"actuals.ParseBalanceTable: row %d: 取得価額 %d が課税口座の残高 %d を上回っている",
				row+1, value.Basis, value.Taxable)
		}
		if i, ok := table.ColumnIndex(BalanceLockedColumn); ok {
			if value.Locked, err = money.ParseYen(fields[i]); err != nil {
				return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: row %d: %w", row+1, err)
			}
		}
		if i, ok := table.ColumnIndex(BalancePartialColumn); ok {
			value.Partial = PartialFlag(fields[i]) == PartialYes
		}
		rows = append(rows, relation.Row[Balance]{Year: year, Value: value})
	}

	if len(rows) == 0 {
		return BalanceTable{}, fmt.Errorf("actuals.ParseBalanceTable: no rows, so the plan has no starting point")
	}
	return BalanceTable{byYear: relation.New(rows)}, nil
}

func (t BalanceTable) Latest() (date.Year, Balance, bool) {
	rows := t.byYear.Rows()
	if len(rows) == 0 {
		return 0, Balance{}, false
	}
	last := rows[len(rows)-1]
	return last.Year, last.Value, true
}

func (t BalanceTable) At(y date.Year) (Balance, bool) { return t.byYear.At(y) }

func (t BalanceTable) Years() []date.Year { return t.byYear.Years() }

const (
	BankYearColumn    tsv.ColumnName = "西暦"
	BankAccountColumn tsv.ColumnName = "口座"
	BankBalanceColumn tsv.ColumnName = "残高[円]"

	BankStateColumn  tsv.ColumnName = "状態"
	BankSourceColumn tsv.ColumnName = "出典"
)

const (
	BankBalanceWritten = "残高"

	BankNotOpened = "未開設"

	BankNotFetched = "未取得"

	BankClosed = "解約"
)

func BankStates() []string {
	return []string{BankBalanceWritten, BankNotOpened, BankNotFetched, BankClosed}
}
