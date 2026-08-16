package table_test

import (
	"reflect"
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/table"
)

func TestABusinessLossShouldComeOffTheOrdinaryIncome(t *testing.T) {
	loss := table.Pay{BusinessReceipts: 400_000, BusinessExpenses: 1_000_000}

	if got, want := loss.BusinessIncome(), money.Yen(-600_000); got != want {
		t.Fatalf("事業所得 = %d, want %d。0 で止めると損益通算にならない", got, want)
	}
}

func TestTheOnlyIncomesInReachAreOrdinaryIncome(t *testing.T) {
	ordinary := map[string]string{
		"SalaryIncome":        "給与所得",
		"BusinessIncome":      "事業所得",
		"MiscellaneousIncome": "雑所得（業務）",
		"PensionIncome":       "雑所得（公的年金等）",
	}

	notIncome := []string{
		"Salary", "Bonus", "SalaryIncomeAdjustment",
		"BusinessReceipts", "BusinessExpenses", "MiscellaneousReceipts",
		"PensionReceived", "PensionBasic", "PensionProportional", "PensionSupplement",
		"OldAgePensionBenefit", "ChildcareLeaveBenefit", "BusinessDeduction",
		"Total", "TotalIncome",
	}

	for _, field := range moneyFieldsOf[table.IncomeRow]() {
		if _, ok := ordinary[field]; ok {
			continue
		}
		if contains(notIncome, field) {
			continue
		}
		t.Errorf("table.IncomeRow に %q という金額の欄が増えている。"+
			"それが所得の種類なら、経常所得（施行令第百九十八条第一号）かどうかを決めること——"+
			"譲渡所得や一時所得なら、損益通算の順序（同条第三号以降）が要る",
			field)
	}
}

func contains(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func moneyFieldsOf[T any]() []string {
	var zero T
	typ := reflect.TypeOf(zero)

	var names []string
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type == reflect.TypeOf(money.Yen(0)) {
			names = append(names, field.Name)
		}
	}
	return names
}
