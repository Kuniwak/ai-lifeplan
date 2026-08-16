package plan

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

const YearColumn tsv.ColumnName = "西暦"

type TableName string

func (p *Plan) Tables() map[TableName]*tsv.Table {
	out := map[TableName]*tsv.Table{
		"timeline": render(p.Timeline,
			[]tsv.ColumnName{"総収入", "総支出", "社会保険料", "税", "収支", "手取り"},
			func(r table.TimelineRow) []string {
				return amounts(r.Receipts, r.Spending, r.SocialInsurance, r.Tax, r.Balance, r.TakeHome())
			}),

		AssetsTable: render(p.Assets,
			[]tsv.ColumnName{"貯蓄", "金融資産", "金融資産(NISA)", "金融資産(課税)", "金融資産(課税)の取得価額",
				"含み益", "含み益に埋まっている税", "NISA満了による払出",
				"年金資産", AssetsTotalColumn, "手が届く額", "収支", "積立", "取崩", "譲渡益への税",
				"掛金", "年金受取", "年金への税", "運用益", "暴落",
				"最後の手段による現金", "借入の利息", "住み替えの差額", AssetsShortfallColumn},
			func(r table.AssetsRow) []string {
				return amounts(r.Cash, r.Invested, r.NISA, r.Taxable, r.Basis,
					r.UnrealisedGain(), r.DeferredTax(), r.MaturedFromNISA,
					r.Locked, r.Total, r.Available(), r.Balance,
					r.Contributed, r.Withdrawn, r.InvestmentTax,
					r.Paid, r.PensionReceived, r.PensionTax,
					r.Returns, r.Crash, r.Raised, r.Interest, r.Housing, r.Shortfall)
			}),

		"outturn": render(p.Outturn,
			[]tsv.ColumnName{"総資産の増減", "手取り", "手取りの基準", "支出−運用損益",
				"収支明細", "説明のつかない差", "一部未記録"},
			func(r actuals.Outturn) []string {
				fields := amounts(r.Movement, r.TakeHome)
				fields = append(fields, string(r.TakeHomeBasis))
				fields = append(fields, amounts(r.Outgoing, r.Cashflow, r.Unexplained)...)
				return append(fields, charged(r.Partial))
			}),

		"real": p.realTable(),

		"expense": render(p.Expense, ExpenseColumns,
			func(r table.ExpenseRow) []string {
				return amounts(r.Living, r.CoupleLiving, r.ChildLiving, r.Medical, r.Allowance, r.Extraordinary,
					r.Education, r.Insurance, r.Housing, r.Rent, r.Deposit, r.LoanPaid, r.Maintenance, r.Total, r.Recurring)
			}),

		"social-insurance": render(p.SocialInsurance,
			[]tsv.ColumnName{"健康保険", "介護保険", "厚生年金", "雇用保険", "国民健康保険", "後期高齢者医療",
				"介護保険(第1号)", "国民年金", "合計"},
			func(r table.SocialInsuranceTotalRow) []string {
				return amounts(r.Health, r.NursingCare, r.Pension, r.EmploymentInsurance,
					r.Kokuho, r.Kouki, r.NursingCareFirstCategory, r.NationalPension, r.Total)
			}),

		"tax": render(p.Tax,
			[]tsv.ColumnName{"所得税", "住民税", "固定資産税", "都市計画税", "合計"},
			func(r table.TaxTotalRow) []string {
				return amounts(r.IncomeTax, r.ResidentTax, r.PropertyTax, r.CityPlanningTax, r.Total)
			}),

		"loan": render(p.Loan,
			[]tsv.ColumnName{"返済額", "利息", "元金", "年末残高"},
			func(r table.LoanYear) []string {
				return amounts(r.Paid, r.Interest, r.Repaid, r.Balance)
			}),

		"child-allowance": render(p.ChildAllowance,
			[]tsv.ColumnName{"児童手当", "所得制限限度額", "所得上限限度額"},
			func(r table.ChildAllowanceRow) []string {
				if r.Limits == nil {
					return []string{yen(r.Total), "", ""}
				}
				return amounts(r.Total, r.Limits.IncomeLimit, r.Limits.IncomeCeiling)
			}),

		"property-tax": render(p.PropertyTax,
			[]tsv.ColumnName{"固定資産税", "都市計画税", "軽減税額", "土地課税標準額", "家屋課税標準額"},
			func(r table.PropertyTaxRow) []string {
				return amounts(r.PropertyTax, r.CityPlanningTax, r.NewHouseRelief,
					r.LandBaseProperty, r.HouseBase)
			}),
	}

	for person, income := range p.Income {
		out[TableName("income-"+romanised(person))] = render(income,
			[]tsv.ColumnName{"給与収入", "給与所得", "所得金額調整控除", "事業収入", "事業必要経費", "業務雑収入",
				"年金(基礎)", "年金(報酬比例)", "年金(加給)", "年金収入", "年金所得", "総収入", "合計所得"},
			func(r table.IncomeRow) []string {
				return amounts(r.Salary, r.SalaryIncome, r.SalaryIncomeAdjustment, r.BusinessReceipts,
					r.BusinessExpenses, r.MiscellaneousReceipts,
					r.PensionBasic, r.PensionProportional, r.PensionSupplement,
					r.PensionReceived, r.PensionIncome, r.Total, r.TotalIncome)
			})
	}
	for person, tax := range p.IncomeTax {
		out[TableName("income-tax-"+romanised(person))] = render(tax,
			[]tsv.ColumnName{"合計所得", "所得控除", "課税所得金額", "所得税額", "住宅ローン控除", "定額減税", "基準所得税額", "復興特別所得税", "申告納税額"},
			func(r table.IncomeTaxRow) []string {
				return amounts(r.TotalIncome, r.Deductions.Total().IncomeTax, r.Taxable, r.Tax,
					r.HousingLoanCredit, r.SpecialCredit, r.BaseTax, r.Surtax, r.Payable)
			})
	}
	for person, tax := range p.ResidentTax {
		out[TableName("resident-tax-"+romanised(person))] = render(tax,
			[]tsv.ColumnName{"合計所得", "課税所得金額", "均等割の課税", "所得割の課税", "均等割", "森林環境税",
				"調整控除", "定額減税", "県民税所得割", "市民税所得割", "住民税"},
			func(r table.ResidentTaxRow) []string {
				fields := amounts(r.TotalIncome, r.Taxable)
				fields = append(fields, charged(r.Liable.PerCapita), charged(r.Liable.Income))
				return append(fields, amounts(r.PerCapita, r.ForestEnvironmentTax, r.Adjustment,
					r.SpecialCredit, r.PrefecturalIncome, r.MunicipalIncome, r.Total)...)
			})
	}

	return out
}

func (p *Plan) realTable() *tsv.Table {
	out := &tsv.Table{Header: []tsv.ColumnName{
		YearColumn, "物価", "資産合計", "含み益に埋まっている税", "手が届く額", "総収入", "総支出",
		"社会保険料", "税", "手取り", "不足",
	}}

	deflated := relation.Join(p.PriceLevels, p.Assets,
		func(_ date.Year, level money.Factor, assets table.AssetsRow) realRow {
			return realRow{Level: level, Assets: assets}
		})
	joined := relation.Join(deflated, p.Timeline,
		func(_ date.Year, r realRow, timeline table.TimelineRow) realRow {
			r.Timeline = timeline
			return r
		})

	for _, row := range joined.Rows() {
		r := row.Value
		fields := []string{fmt.Sprint(row.Year), r.Level.String()}
		fields = append(fields, amounts(
			r.Level.Deflate(r.Assets.Total), r.Level.Deflate(r.Assets.DeferredTax()),
			r.Level.Deflate(r.Assets.Available()),
			r.Level.Deflate(r.Timeline.Receipts), r.Level.Deflate(r.Timeline.Spending),
			r.Level.Deflate(r.Timeline.SocialInsurance), r.Level.Deflate(r.Timeline.Tax),
			r.Level.Deflate(r.Timeline.TakeHome()), r.Level.Deflate(r.Assets.Shortfall),
		)...)
		out.Rows = append(out.Rows, fields)
	}
	return out
}

type realRow struct {
	Level    money.Factor
	Assets   table.AssetsRow
	Timeline table.TimelineRow
}

func yen(amount money.Yen) string { return fmt.Sprint(int64(amount)) }

func charged(yes bool) string {
	if yes {
		return string(law.DesignatedCityYes)
	}
	return "いいえ"
}

func romanised(person table.PersonName) string {
	switch person {
	case Earner:
		return "husband"
	case Spouse:
		return "wife"
	default:
		return strings.ToLower(string(person))
	}
}

func render[T any](built relation.Table[T], columns []tsv.ColumnName, of func(T) []string) *tsv.Table {
	out := &tsv.Table{Header: append([]tsv.ColumnName{YearColumn}, columns...)}
	for _, row := range built.Rows() {
		fields := make([]string, 0, len(columns)+1)
		fields = append(fields, fmt.Sprint(row.Year))
		fields = append(fields, of(row.Value)...)
		out.Rows = append(out.Rows, fields)
	}
	return out
}

func amounts(of ...money.Yen) []string {
	fields := make([]string, 0, len(of))
	for _, amount := range of {
		fields = append(fields, yen(amount))
	}
	return fields
}

func (p *Plan) WriteTables(dir string) error {
	for name, built := range p.Tables() {
		if err := tsv.WriteFile(filepath.Join(dir, string(name)+".tsv"), built); err != nil {
			return err
		}
	}
	return nil
}

var ExpenseColumns = []tsv.ColumnName{
	"生活費合計", "夫婦生活費", "子生活費", "医療費", "小遣い", "臨時費用",
	"教育費合計", "保険料合計", "住宅合計", "家賃", "頭金", "ローン返済", "住宅維持費", "費目合計",
	"経常支出",
}
