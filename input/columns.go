package input

import (
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	PersonColumn       tsv.ColumnName = "人"
	RelationColumn     tsv.ColumnName = "続柄"
	BornOnColumn       tsv.ColumnName = "生年月日"
	StageColumn        tsv.ColumnName = "就学段階"
	StageFromAgeColumn tsv.ColumnName = "開始満年齢"
	MunicipalityColumn tsv.ColumnName = "自治体"
	BoughtInColumn     tsv.ColumnName = "取得年"
)

const (
	AnnualSalaryColumn tsv.ColumnName = "給与収入[円/年]"
	BonusColumn        tsv.ColumnName = "賞与収入[円/年]"
	BonusesAYearColumn tsv.ColumnName = "賞与回数[回/年]"
	LeaveMonthsColumn  tsv.ColumnName = "育休月数[月]"

	WeeklyHoursColumn tsv.ColumnName = "週所定労働時間[時間/週]"

	NormalWeeklyHoursColumn tsv.ColumnName = "通常労働者週所定労働時間[時間/週]"

	SpecifiedWorkplaceColumn tsv.ColumnName = "特定適用事業所区分"

	ExemptMonthsColumn tsv.ColumnName = "保険料免除月[月]"

	BusinessReceiptsColumn tsv.ColumnName = "事業収入[円/年]"

	SmallDepreciableColumn tsv.ColumnName = "少額減価償却資産取得価額[円/年]"

	BusinessExpensesColumn tsv.ColumnName = "事業必要経費[円/年]"

	MiscellaneousReceiptsColumn tsv.ColumnName = "業務雑収入[円/年]"

	BlueFormRecordKeepingColumn tsv.ColumnName = "青色申告区分"

	PensionStartColumn tsv.ColumnName = "受給開始年"

	PensionExpectedColumn tsv.ColumnName = "支給率"
)

const (
	CoupleLivingColumn  tsv.ColumnName = "生活費[円/月]"
	ChildLivingColumn   tsv.ColumnName = "生活費[円/年]"
	TuitionColumn       tsv.ColumnName = "学費[円/年]"
	AllowanceColumn     tsv.ColumnName = "小遣い[円/月]"
	MedicalColumn       tsv.ColumnName = "医療費[円]"
	MedicalRefundColumn tsv.ColumnName = "保険で補填された金額[円]"
	ExtraordinaryColumn tsv.ColumnName = "費用[円]"
	LifeInsuranceColumn tsv.ColumnName = "生命保険料[円/年]"

	MedicalInsuranceColumn tsv.ColumnName = "介護医療保険料[円/年]"
	FireInsuranceColumn    tsv.ColumnName = "火災保険料[円]"
	QuakeInsuranceColumn   tsv.ColumnName = "地震保険料[円]"

	InsuranceTermColumn tsv.ColumnName = "保険期間[年]"
	RentColumn          tsv.ColumnName = "家賃[円/年]"
	MaintenanceColumn   tsv.ColumnName = "修繕費[円]"

	MeasureColumn      tsv.ColumnName = "手段"
	MeasureFromAge     tsv.ColumnName = "使える最低年齢"
	MeasureProceedRate tsv.ColumnName = "受取率"
	MeasureInterest    tsv.ColumnName = "年利"
	MeasureRentMonthly tsv.ColumnName = "家賃[円/月]"
	MeasureGivesUpHome tsv.ColumnName = "手放すか"
	DepositColumn      tsv.ColumnName = "頭金[円]"

	MutualAidContributionColumn tsv.ColumnName = "小規模企業共済等掛金[円/年]"
)

const (
	DownPaymentColumn    tsv.ColumnName = "頭金[円]"
	PrincipalColumn      tsv.ColumnName = "借入額[円]"
	LoanYearsColumn      tsv.ColumnName = "借入期間[年]"
	LoanRateColumn       tsv.ColumnName = "固定年利"
	LoanFixedYearsColumn tsv.ColumnName = "固定期間[年]"
	LoanFloatingColumn   tsv.ColumnName = "固定期間後の実質年利"
	DisabledPersonColumn tsv.ColumnName = "人"
	DisabilityCertified  tsv.ColumnName = "認定年"
	DisabilityCategory   tsv.ColumnName = "区分"

	DisabilityPensionColumn tsv.ColumnName = "障害厚生年金の受給要件"

	ReturnColumn       tsv.ColumnName = "実質運用利率"
	CrashColumn        tsv.ColumnName = "金融資産下落率"
	LandValueColumn    tsv.ColumnName = "土地評価額[円]"
	HouseBaseColumn    tsv.ColumnName = "家屋課税標準額[円]"
	AssessedYearColumn tsv.ColumnName = "年度"
)

var BlueFormRecordKeepingWords = []string{
	"白色申告",
	"青色申告（簡易）",
	"青色申告（複式）",
	"青色申告（複式・電子帳簿保存またはe-Tax）",
}

var DisabilityPensionWords = []string{"はい", "いいえ"}

const (
	SpecifiedWorkplace    = "該当"
	NotSpecifiedWorkplace = "非該当"
)

var SpecifiedWorkplaceWords = []string{SpecifiedWorkplace, NotSpecifiedWorkplace}

var RelationWords = []string{"子", "配偶者", "本人"}

const (
	LoanNameColumn       tsv.ColumnName = "契約"
	LoanDrawnInColumn    tsv.ColumnName = "借入年"
	LoanFirstYearColumn  tsv.ColumnName = "初回返済年"
	LoanFirstMonthColumn tsv.ColumnName = "初回返済月"
)

const SellNISAFirstColumn tsv.ColumnName = "先に売る口座"

type SellFirst string

const (
	SellNISA    SellFirst = "NISA"
	SellTaxable SellFirst = "課税"
)

const SourceColumn tsv.ColumnName = "出典"

const SmallDepreciableYearlyLimit money.Yen = 3_000_000
