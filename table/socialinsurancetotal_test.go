package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func socialInsuranceTotalOfTheBaseProject(t *testing.T) relation.Table[table.SocialInsuranceTotalRow] {
	t.Helper()

	built, err := table.SocialInsuranceTotalTable(table.SocialInsuranceTotalInput{
		Employees: map[table.PersonName]relation.Table[table.SocialInsuranceRow]{
			"夫": socialInsuranceOfTheBaseProject(t),
		},
		Household: householdInsuranceOfTheBaseProject(t),
	})
	if err != nil {
		t.Fatalf("table.SocialInsuranceTotalTable: %v", err)
	}
	return built
}
