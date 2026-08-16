package actuals

import (
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

const (
	HoldingsPath    tsv.Slot = "actuals/securities/holdings.tsv"
	TransactionPath tsv.Slot = "actuals/securities/transaction.tsv"
	OutsidePath     tsv.Slot = "actuals/securities/outside.tsv"

	WifeHoldingsPath tsv.Slot = "actuals/securities/wife-holdings.tsv"
)

const (
	StatementsFrom    = "2024-03-31"
	StatementsThrough = "2025-12-31"
)

const (
	OutsideYearColumn   tsv.ColumnName = "西暦"
	OutsideAmountColumn tsv.ColumnName = "報告書の外の金融資産[円]"

	OutsidePocketColumn tsv.ColumnName = "口座種別"
)

const (
	WifeHoldingAsOfColumn   tsv.ColumnName = "基準日"
	WifeHoldingPocketColumn tsv.ColumnName = "口座種別"
	WifeHoldingValueColumn  tsv.ColumnName = "時価評価額[円]"
)

const (
	HoldingAsOfColumn   tsv.ColumnName = "基準日"
	HoldingFundColumn   tsv.ColumnName = "ファンド"
	HoldingPocketColumn tsv.ColumnName = "口座種別"
	HoldingUnitsColumn  tsv.ColumnName = "数量[口]"
	HoldingPriceColumn  tsv.ColumnName = "時価単価[円/万口]"
	HoldingValueColumn  tsv.ColumnName = "時価評価額[円]"
	HoldingBookColumn   tsv.ColumnName = "簿価単価[円/万口]"
	HoldingGainColumn   tsv.ColumnName = "評価損益額[円]"

	HoldingBasisColumn tsv.ColumnName = "個別元本[円/万口]"
)

const (
	TradeDateColumn    tsv.ColumnName = "約定日"
	TradeFundColumn    tsv.ColumnName = "ファンド"
	TradeDepositColumn tsv.ColumnName = "預り区分"
	TradeDealColumn    tsv.ColumnName = "取引"
	TradeUnitsColumn   tsv.ColumnName = "数量[口]"
)

type (
	Pocket string

	Deposit string

	Deal string
)

const (
	NISAPocket    Pocket = "ＮＩＳＡ"
	TaxablePocket Pocket = "特定口座"

	NISADeposit    Deposit = "ＮＩＳＡ預り"
	TaxableDeposit Deposit = "特定預り"

	Bought Deal = "購入"
	Sold   Deal = "解約"
)

var Pockets = map[Deposit]Pocket{
	NISADeposit:    NISAPocket,
	TaxableDeposit: TaxablePocket,
}

func LedgerVocabulary() validate.LedgerVocabulary {
	pockets := make(map[string]string, len(Pockets))
	for deposit, pocket := range Pockets {
		pockets[string(deposit)] = string(pocket)
	}
	return validate.LedgerVocabulary{
		Pockets: pockets,
		Bought:  string(Bought),
		Sold:    string(Sold),
	}
}

const OldNISABoughtIn = 2023

func BalancePockets() validate.BalancePockets {
	return validate.BalancePockets{NISA: string(NISAPocket), Taxable: string(TaxablePocket)}
}
