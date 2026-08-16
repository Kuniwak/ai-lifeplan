package law

import (
	"github.com/Kuniwak/lifeplan/panictest"
	"os"
	"testing"
)

func employmentInsuranceTable(t *testing.T) EmploymentInsuranceTable {
	t.Helper()

	parsed := MustLoadEmploymentInsuranceRates(t, os.DirFS("../"+LawDirectory))
	return EmploymentInsuranceTable{YearRateTable: parsed}
}

func TestEmploymentInsurancePremiumShouldRefuseAYearBeforeTheTableStarts(t *testing.T) {
	refused := panictest.Recovered(func() {
		employmentInsuranceTable(t).Premium(3_000_000, 1995)
	})
	if refused == nil {
		t.Fatal("表が始まるより前の年に答えている。誰も書いていない年である")
	}
}
