package actuals

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

var UnclassifiedLimit = money.NewRate(2, 100)

const YearPrefix = 4

func knownItemNames() []string {
	names := make([]string, 0, len(KnownItems))
	for _, item := range KnownItems {
		names = append(names, string(item))
	}
	return names
}

var KnownItems = []Item{
	"給与収入", "事業収入", "業務雑収入", "その他収入",
	"夫婦生活費", "子1生活費", "子2生活費", "子3生活費", "子生活費（共通）",
	"医療費", "夫小遣い", "妻小遣い", "臨時費用",
	"学費", "生命保険料", "損害保険料", "事業経費",
	"家賃", "住宅ローン返済", "住宅維持費", "税",
	Unclassified, UnseenCash, UnseenCard,
}

func bankStates() validate.AmountStates {
	return validate.AmountStates{
		Written:   BankBalanceWritten,
		Absent:    []string{BankNotOpened, BankClosed},
		Unfetched: []string{BankNotFetched},
	}
}

func unfetchedBankYears() validate.LowerBoundYears {
	return validate.LowerBoundYears{
		Needs: []tsv.Slot{BankBalancePath},
		Years: func(tables map[tsv.Slot]*tsv.Table) []string {
			banks := tables[BankBalancePath]
			atYear, hasYear := banks.ColumnIndex(BankYearColumn)
			atState, hasState := banks.ColumnIndex(BankStateColumn)
			if !hasYear || !hasState {
				return nil
			}

			unfetched := bankStates().Unfetched
			var years []string
			for _, fields := range banks.Rows {
				if slices.Contains(unfetched, fields[atState]) && !slices.Contains(years, fields[atYear]) {
					years = append(years, fields[atYear])
				}
			}
			return years
		},
	}
}

func (b Buckets) KnownKinds() validate.KnownKinds {
	column := map[AssetKind]tsv.ColumnName{
		Cash:     BalanceCashColumn,
		Invested: BalanceInvestedColumn,
		Locked:   BalanceLockedColumn,
	}
	kinds := make(validate.KnownKinds, len(b.byKind))
	for kind, bucket := range b.byKind {
		kinds[kind] = validate.KnownKind{
			Column:       column[bucket],
			RecordedFrom: b.knownFrom[kind],
		}
	}
	return kinds
}

var YearsOutsideComparison = []string{"2022"}

var BankYearsOutsideComparison = []string{}

func Rules(root string) ([]validate.Rule, map[tsv.Slot]*tsv.Table, error) {
	tables := make(map[tsv.Slot]*tsv.Table, 6)
	for _, path := range []tsv.Slot{
		CashflowPath, AdjustmentsPath, BalancePath,
		HoldingsPath, TransactionPath, OutsidePath, BankBalancePath, WifeHoldingsPath,
		KnownPath, SourcesPath, BankAccountsPath,
	} {
		table, err := tsv.ReadFile(tsv.Under(root, string(path)))
		if err != nil {
			return nil, nil, fmt.Errorf("actuals: %w", err)
		}
		tables[path] = table
	}

	accounts, err := tsv.ReadFile(tsv.Under(root, string(AccountsPath)))
	if err != nil {
		return nil, nil, fmt.Errorf("actuals: %w", err)
	}
	buckets, err := ParseBuckets(accounts)
	if err != nil {
		return nil, nil, err
	}

	rules := []validate.Rule{
		validate.Scoped(validate.Unclassified(
			CashflowPath, CashflowItemColumn, CashflowAmountColumn,
			string(Unclassified), UnclassifiedLimit, CashflowMonthColumn, YearPrefix,
		), CashflowPath),
		validate.Scoped(validate.ItemsKnown(
			CashflowPath, CashflowItemColumn, knownItemNames(),
		), CashflowPath),
		validate.Scoped(validate.UniqueKey(
			CashflowPath, []tsv.ColumnName{CashflowMonthColumn, CashflowItemColumn},
		), CashflowPath),

		validate.Scoped(validate.YearsAreCovered(
			SourcesPath, SourceFileColumn,
			CashflowPath, CashflowMonthColumn,
		), CashflowPath),

		validate.Scoped(validate.YearsOutsideComparison(
			CashflowPath, CashflowMonthColumn,
			BalancePath, BalanceYearColumn,
			validate.AgainstMovement, YearsOutsideComparison,
		), CashflowPath),

		validate.Scoped(validate.ItemsKnown(
			AdjustmentsPath, AdjustmentItemColumn, knownItemNames(),
		), AdjustmentsPath),
		validate.Scoped(validate.UniqueKey(
			AdjustmentsPath,
			[]tsv.ColumnName{AdjustmentMonthColumn, AdjustmentItemColumn},
		), AdjustmentsPath),

		validate.Scoped(validate.BalanceFollowsTheBank(
			BalancePath, BalanceYearColumn, BalanceCashColumn,
			BankBalancePath, BankYearColumn, BankBalanceColumn,
		), BalancePath),
		validate.Scoped(validate.KeysCoverYears(
			BankBalancePath, BankYearColumn, BankAccountColumn,
			BalancePath, BalanceYearColumn,
		), BankBalancePath),

		validate.Scoped(validate.KeysAreDeclared(
			BankBalancePath, BankAccountColumn,
			BankAccountsPath, BankAccountColumn,
		), BankBalancePath),

		validate.Scoped(validate.YearsOutsideComparison(
			BankBalancePath, BankYearColumn,
			BalancePath, BalanceYearColumn,
			validate.AgainstLevel, BankYearsOutsideComparison,
		), BankBalancePath),

		validate.Scoped(validate.ColumnSchema(BankAccountsPath, []validate.Column{
			{Name: BankAccountColumn, Unit: validate.UnitText, Parse: validate.AsText},
			{Name: BankSourceColumn, Unit: validate.UnitText, Parse: validate.AsText},
		}), BankAccountsPath),

		validate.Scoped(validate.UniqueKey(BankAccountsPath,
			[]tsv.ColumnName{BankAccountColumn}), BankAccountsPath),

		validate.Scoped(validate.BalanceFollowsTheKnown(
			BalancePath, BalanceYearColumn,
			validate.PartialMark{Column: BalancePartialColumn, Yes: string(PartialYes)},
			KnownPath, KnownYearColumn, KnownKindColumn, KnownBalanceColumn,
			buckets.KnownKinds(), unfetchedBankYears(),
		), BalancePath),

		validate.Scoped(validate.ColumnSchema(BankBalancePath, []validate.Column{
			{Name: BankYearColumn, Unit: validate.UnitText, Parse: validate.AsYear},
			{Name: BankAccountColumn, Unit: validate.UnitText, Parse: validate.AsText},
			{Name: BankBalanceColumn, Unit: validate.UnitYen, Parse: validate.AsOptional(validate.AsYen)},
			{Name: BankStateColumn, Unit: validate.UnitText, Parse: validate.AsOneOf(BankStates()...)},
			{Name: BankSourceColumn, Unit: validate.UnitText, Parse: validate.AsText},
		}), BankBalancePath),

		validate.Scoped(validate.AmountAgreesWithItsState(
			BankBalancePath, BankBalanceColumn, BankStateColumn,
			bankStates(),
		), BankBalancePath),

		validate.Scoped(validate.StateOnlyAtTheStart(
			BankBalancePath, BankAccountColumn, BankYearColumn, BankStateColumn,
			BankNotOpened,
		), BankBalancePath),

		validate.Scoped(validate.StateOnlyAtTheEnd(
			BankBalancePath, BankAccountColumn, BankYearColumn, BankStateColumn,
			BankClosed,
		), BankBalancePath),
		validate.Scoped(validate.UniqueKey(
			BankBalancePath, []tsv.ColumnName{BankYearColumn, BankAccountColumn},
		), BankBalancePath),

		validate.Scoped(validate.AmountsAddUp(
			BalancePath, BalanceTotalColumn,
			[]tsv.ColumnName{BalanceCashColumn, BalanceInvestedColumn, BalanceLockedColumn},
		), BalancePath),
		validate.Scoped(validate.UniqueKey(
			BalancePath, []tsv.ColumnName{BalanceYearColumn},
		), BalancePath),
		validate.Scoped(validate.AmountsAddUp(
			BalancePath, BalanceInvestedColumn,
			[]tsv.ColumnName{BalanceNISAColumn, BalanceTaxableColumn},
		), BalancePath),
		validate.Scoped(validate.BalanceFollowsTheStatements(
			validate.SplitBalanceSide{
				Slot: BalancePath, Year: BalanceYearColumn, NISA: BalanceNISAColumn,
				Taxable: BalanceTaxableColumn, Basis: BalanceBasisColumn,
			},
			validate.StatementSide{
				Slot: HoldingsPath, AsOf: HoldingAsOfColumn, Pocket: HoldingPocketColumn,
				Value: HoldingValueColumn, Gain: HoldingGainColumn,
			},
			validate.OutsideSide{
				Slot: OutsidePath, Year: OutsideYearColumn,
				Amount: OutsideAmountColumn, Pocket: OutsidePocketColumn,
			},
			BalancePockets(),
		), BalancePath),

		validate.Scoped(validate.UniqueKey(
			OutsidePath, []tsv.ColumnName{OutsideYearColumn, OutsidePocketColumn},
		), OutsidePath),

		validate.Scoped(validate.OutsideFollowsTheHoldings(
			OutsidePath, OutsideYearColumn, OutsideAmountColumn, OutsidePocketColumn,
			WifeHoldingsPath, WifeHoldingAsOfColumn, WifeHoldingPocketColumn, WifeHoldingValueColumn,
		), OutsidePath),

		validate.Scoped(validate.StatementsCoverEveryQuarter(
			HoldingsPath, HoldingAsOfColumn,
			StatementsFrom, StatementsThrough,
		), HoldingsPath),
		validate.Scoped(validate.HoldingsFollowTheLedger(
			validate.HoldingSide{
				Slot: HoldingsPath, AsOf: HoldingAsOfColumn, Fund: HoldingFundColumn,
				Pocket: HoldingPocketColumn, Units: HoldingUnitsColumn,
			},
			validate.LedgerSide{
				Slot: TransactionPath, Traded: TradeDateColumn, Fund: TradeFundColumn,
				Deposit: TradeDepositColumn, Deal: TradeDealColumn, Units: TradeUnitsColumn,
			},
			LedgerVocabulary(),
		), HoldingsPath),
		validate.Scoped(validate.HoldingsValueFollowsThePrices(
			HoldingsPath, HoldingUnitsColumn, HoldingPriceColumn,
			HoldingValueColumn, HoldingBookColumn, HoldingGainColumn,
		), HoldingsPath),
	}
	return rules, tables, nil
}
