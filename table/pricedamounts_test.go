package table_test

import (
	"reflect"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func everyWrittenAmountIsSomething(levels map[input.PricedItem]relation.Table[money.Factor]) table.ExpenseInput {
	years := []date.Year{2000, 2001}
	constant := func(v money.Yen) relation.Table[money.Yen] { return relation.Constant(years, v) }
	byYear := func(v money.Yen) map[date.Year]money.Yen {
		return map[date.Year]money.Yen{2000: v, 2001: v}
	}

	return table.ExpenseInput{
		Calendar: relation.Constant(years, table.CalendarRow{
			Ages: []table.PersonYear{
				{Name: "夫", Age: 40},
				{Name: "子", Age: 10, Stage: "小学校", Relation: table.Child},
			},
		}),
		CoupleLivingMonthly: constant(101_000),
		ChildLivingByStage:  map[table.Stage]money.Yen{"小学校": 102_000},
		TuitionByStage:      map[table.Stage]money.Yen{"小学校": 103_000},
		AllowanceMonthly: map[table.PersonName]relation.Table[money.Yen]{
			"夫": constant(104_000),
			"子": constant(105_000),
		},
		MedicalPaid:         constant(106_000),
		MedicalRefunded:     constant(1_000),
		Extraordinary:       byYear(107_000),
		FireInsurance:       byYear(108_000),
		EarthquakeInsurance: byYear(109_000),
		InsuranceTerm:       byYear(5),
		Maintenance:         byYear(110_000),
		Deposit:             byYear(111_000),
		LifeInsurance:       constant(112_000),
		MedicalInsurance:    constant(8_580),
		Rent:                constant(113_000),
		Loan:                relation.Constant(years, table.LoanYear{Paid: 114_000}),
		PriceLevelsByItem:   levels,
	}
}

func TestEveryAmountShouldPassThroughTheInflatorIntoTheYearsMoney(t *testing.T) {
	ratios := make(map[input.PricedItem]money.PriceMove, len(input.PricedItems))
	for i, item := range input.PricedItems {
		ratios[item] = money.RatioMove(money.NewRate(int64(i+1)*100, 100))
	}

	years := []date.Year{2000, 2001}
	flat, err := table.PriceLevelsByItem(relation.Constant(years, money.NewRate(0, 100)), ratios)
	if err != nil {
		t.Fatalf("table.PriceLevelsByItem: %v", err)
	}
	levels, err := table.PriceLevelsByItem(relation.Constant(years, money.NewRate(100, 100)), ratios)
	if err != nil {
		t.Fatalf("table.PriceLevelsByItem: %v", err)
	}

	still, err := table.ExpenseTable(everyWrittenAmountIsSomething(flat))
	if err != nil {
		t.Fatalf("物価が動かないほう: %v", err)
	}
	moved, err := table.ExpenseTable(everyWrittenAmountIsSomething(levels))
	if err != nil {
		t.Fatalf("物価が動くほう: %v", err)
	}

	const aTotal input.PricedItem = ""
	pricedBy := map[string]input.PricedItem{
		"CoupleLiving":         input.CoupleLivingItem,
		"ChildLiving":          input.ChildLivingItem,
		"Medical":              input.MedicalItem,
		"MedicalPaid":          input.MedicalItem,
		"MedicalRefunded":      input.MedicalItem,
		"Allowance":            input.AllowanceItem,
		"Extraordinary":        input.ExtraordinaryItem,
		"Education":            input.EducationItem,
		"Life":                 input.LifeInsuranceItem,
		"MedicalCover":         input.LifeInsuranceItem,
		"EarthquakeDeductible": input.QuakeInsuranceItem,
		"Fire":                 input.FireInsuranceItem,
		"Earthquake":           input.QuakeInsuranceItem,
		"Rent":                 input.RentItem,
		"Deposit":              input.DepositItem,
		"LoanPaid":             input.LoanPaidItem,
		"Maintenance":          input.MaintenanceItem,

		"Living":    aTotal,
		"Insurance": aTotal,
		"Housing":   aTotal,
		"Total":     aTotal,
		"Recurring": aTotal,
	}

	const y = date.Year(2001)
	before, _ := still.At(y)
	after, _ := moved.At(y)

	typ := reflect.TypeOf(table.ExpenseRow{})
	yen := reflect.TypeOf(money.Yen(0))
	a, b := reflect.ValueOf(before), reflect.ValueOf(after)

	checked := 0
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type != yen {
			continue
		}

		item, named := pricedBy[field.Name]
		if !named {
			t.Errorf("%s がどの項目の答えで動くのか、誰も言っていない。"+
				"pricedBy に足すこと——合計なら aTotal を書く", field.Name)
			continue
		}

		was, now := a.Field(i).Interface().(money.Yen), b.Field(i).Interface().(money.Yen)
		if was == 0 {
			t.Errorf("%s が %d 年に 0 である。この入力はその項目を動かしていないので、"+
				"物価を通ったかどうかを何も言えない。everyWrittenAmountIsSomething に額を足すこと",
				field.Name, y)
			continue
		}
		if item == aTotal {
			if now == was {
				t.Errorf("%s: 部品が動いたのに合計が %d のまま", field.Name, was)
			}
			checked++
			continue
		}

		byYear, ok := levels[item]
		if !ok {
			t.Fatalf("%q に物価が無い", item)
		}
		level, ok := byYear.At(y)
		if !ok {
			t.Fatalf("%q の %d 年の物価が無い", item, y)
		}
		want := level.Apply(was)
		if now != want {
			t.Errorf("%s: %d のはずが %d。%q の答えで動いていない——"+
				"into(…) の項目が違うか、包み忘れている", field.Name, want, now, item)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("ExpenseRow に money.Yen のフィールドが 1 つも無い。この検査は何も見ていない")
	}
}
