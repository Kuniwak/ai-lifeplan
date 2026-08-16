package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/sheets"
)

func theVerificationBlock(t *testing.T) []sheets.VerificationRow {
	t.Helper()

	rows, err := sheets.New(os.DirFS("../testdata/sheets")).Verification()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func verificationWant(t *testing.T, rows []sheets.VerificationRow, name string) (money.Yen, money.Yen) {
	t.Helper()

	for _, r := range rows {
		if r.Name == name {
			return r.Want, r.Within
		}
	}
	t.Fatalf("%q is not in the verification block", name)
	return 0, 0
}
