package law_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/law"
)

func TestAMonthBeforeTotalRemunerationShouldBeRefused(t *testing.T) {
	rates := law.MustLoadPensionRevaluationRates(t, os.DirFS("../"+law.LawDirectory))

	_, err := rates.EarningsRelatedPension(2025, []law.Remuneration{
		{Year: 2002, Month: 4, Amount: 300_000},
	})
	if err == nil {
		t.Fatal("総報酬制より前の月が通った。乗率も平均の取り方も違う")
	}
	if !strings.Contains(err.Error(), "総報酬制") {
		t.Errorf("エラーがそのことを言っていない: %v", err)
	}
}
