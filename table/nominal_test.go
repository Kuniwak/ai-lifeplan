package table_test

import (
	"reflect"
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/table"
)

func TestEveryAmountShouldPassThroughTheWindowIntoNominalMoney(t *testing.T) {
	twice := money.NoInflation().After(money.NewRate(100, 100))

	for _, c := range []struct {
		name    string
		filled  any
		nominal func(any, money.Factor) any
	}{
		{
			name:   "Pay",
			filled: filledWithAmounts(t, table.Pay{}),
			nominal: func(v any, f money.Factor) any {
				return v.(table.Pay).Nominal(f, money.NoInflation())
			},
		},
		{
			name:    "Pension",
			filled:  filledWithAmounts(t, table.Pension{}),
			nominal: func(v any, f money.Factor) any { return v.(table.Pension).Nominal(f) },
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := reflect.ValueOf(c.filled)
			after := reflect.ValueOf(c.nominal(c.filled, twice))

			for i := range before.NumField() {
				field := before.Type().Field(i)
				if field.Type != reflect.TypeOf(money.Yen(0)) {
					continue
				}
				was := before.Field(i).Int()
				is := after.Field(i).Int()
				if is != was*2 {
					t.Errorf("%s.%s が %d から %d にしかならない（2 倍の %d のはず）。"+
						"Nominal に書き足し忘れている", c.name, field.Name, was, is, was*2)
				}
			}
		})
	}
}

func filledWithAmounts(t *testing.T, of any) any {
	t.Helper()

	v := reflect.New(reflect.TypeOf(of)).Elem()
	amount := int64(1_000)
	for i := range v.NumField() {
		if v.Type().Field(i).Type != reflect.TypeOf(money.Yen(0)) {
			continue
		}
		if !v.Field(i).CanSet() {
			t.Fatalf("%s.%s に値を入れられない", v.Type(), v.Type().Field(i).Name)
		}
		v.Field(i).SetInt(amount)
		amount += 1_000
	}
	return v.Interface()
}

func TestRealWageGrowthShouldMoveWagesAndNotBusinessReceipts(t *testing.T) {
	twice := money.NoInflation().After(money.NewRate(100, 100))

	pay := table.Pay{
		Salary:                1_000_000,
		Bonus:                 200_000,
		BusinessReceipts:      300_000,
		MiscellaneousReceipts: 40_000,
	}

	got := pay.Nominal(money.NoInflation(), twice)

	for _, c := range []struct {
		name      string
		got, want money.Yen
	}{
		{name: "給与", got: got.Salary, want: 2_000_000},
		{name: "賞与", got: got.Bonus, want: 400_000},
		{name: "事業収入", got: got.BusinessReceipts, want: 300_000},
		{name: "業務雑収入", got: got.MiscellaneousReceipts, want: 40_000},
	} {
		if c.got != c.want {
			t.Errorf("%s が %d。%d のはず", c.name, c.got, c.want)
		}
	}
}

func TestPricesAndWagesShouldBeComposedBeforeEitherIsApplied(t *testing.T) {
	prices := money.NoInflation().After(money.NewRate(2, 100))
	wages := money.NoInflation().After(money.NewRate(15, 1000))

	pay := table.Pay{Salary: 15}

	if got, want := pay.Nominal(prices, wages).Salary, money.Yen(16); got != want {
		t.Errorf("掛けてから当てると %d。%d のはず", got, want)
	}
	if got, want := prices.Apply(wages.Apply(pay.Salary)), money.Yen(15); got != want {
		t.Fatalf("二度当てると %d。このテストは %d を前提にしている", got, want)
	}
}

func TestEveryPensionAmountShouldPassThroughTheWindowIntoTheScenariosLevel(t *testing.T) {
	half := money.NewRate(50, 100)

	filled := filledWithAmounts(t, table.Pension{}).(table.Pension)
	moved := filled.AtLevel(table.PensionLevel{Basic: half, Proportional: half})

	before := reflect.ValueOf(filled)
	after := reflect.ValueOf(moved)

	for i := range before.NumField() {
		field := before.Type().Field(i)
		if field.Type != reflect.TypeOf(money.Yen(0)) {
			continue
		}
		was, is := before.Field(i).Int(), after.Field(i).Int()
		if is != was/2 {
			t.Errorf("Pension.%s が %d から %d にしかならない（半分の %d のはず）。"+
				"AtLevel に書き足し忘れている", field.Name, was, is, was/2)
		}
	}
}
