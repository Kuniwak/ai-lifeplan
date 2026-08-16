package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
)

func thePayslips(t *testing.T) []actuals.Payslip {
	t.Helper()

	slips, err := actuals.Payslips(os.DirFS("../" + actuals.PayslipDir))
	if err != nil {
		t.Fatalf("actuals.Payslips: %v", err)
	}
	return slips
}

func salaryMonths(t *testing.T) (paid, charged map[monthAt]bool) {
	t.Helper()

	slips, err := actuals.Payslips(os.DirFS("../" + actuals.PayslipDir))
	if err != nil {
		t.Fatalf("actuals.Payslips: %v", err)
	}

	paid, charged = map[monthAt]bool{}, map[monthAt]bool{}
	for _, slip := range slips {
		if slip.Kind != actuals.PayslipSalary {
			continue
		}
		at := monthAt{int(slip.Year), slip.Month, slip.Employer}
		paid[at] = true
		if slip.Health > 0 || slip.Pension > 0 {
			charged[at] = true
		}
	}
	return paid, charged
}

type monthAt struct {
	year, month int
	employer    string
}

func TestTheFirstYearShouldFallBackToItsOwnGrade(t *testing.T) {
	built := socialInsuranceDerivedOfTheBaseProject(t)
	grades, rates := theHealthGradesAndRates(t)
	pay := thePayOfTheBaseProject(t)

	first := pay.Years()[0]

	row, ok := built.At(first)
	if !ok {
		t.Fatalf("%d 年の行が無い", first)
	}

	own, _ := pay.At(first)
	if want := rates.HealthPremium(grades.Lookup(own.Monthly()), first) * date.MonthsAYear; row.HealthOnPay != want {
		t.Errorf("%d 年の給与にかかる健康保険料 = %d、%d のはず（前年が無いので 12 か月とも当年の等級）",
			first, row.HealthOnPay, want)
	}
}
