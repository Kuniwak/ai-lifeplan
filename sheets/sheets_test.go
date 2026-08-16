package sheets

import (
	"os"
	"slices"
	"testing"
)

func fixed(t *testing.T) Copy {
	t.Helper()
	return New(os.DirFS("../testdata/sheets"))
}

func TestEveryFixedTableShouldParse(t *testing.T) {
	copy := fixed(t)
	names, err := copy.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("the fixed copy is empty; testdata/sheets is missing or unreadable")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			table, err := copy.Table(name)
			if err != nil {
				t.Fatalf("Table: %v", err)
			}
			if len(table.Header) == 0 {
				t.Error("the table has no header, so its columns cannot be named")
			}
			for i, row := range table.Rows {
				if len(row) != len(table.Header) {
					t.Errorf("row %d has %d fields but the header has %d", i+1, len(row), len(table.Header))
				}
			}
		})
	}
}

func TestTheCopyShouldHoldTheBlocksTheLawTestsNeed(t *testing.T) {
	wanted := []string{
		"income-tax-brackets", "salary-income-deduction",
		"pension-income-deduction-over65", "pension-income-deduction-under65",
		"resident-tax-income", "resident-tax-pension-income-over65",
		"basic-deduction-income-tax", "basic-deduction-resident-tax",
		"spouse-deduction", "spouse-special-deduction",
		"standard-remuneration-its", "standard-remuneration-kousei",
		"child-allowance-limits", "employment-insurance-rate",
		"life-insurance-deduction", "national-pension-premium",
		"depreciation-rate", "property-tax-schedule",
		"kouki-per-capita", "kouki-rate",
	}

	copy := fixed(t)
	names, err := copy.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}

	for _, want := range wanted {
		if !slices.Contains(names, want) {
			t.Errorf("the fixed copy has no %q", want)
		}
	}
}

func TestTableNG(t *testing.T) {
	copy := fixed(t)

	_, err := copy.Table("no-such-block")

	if err == nil {
		t.Error("want error for a table that is not in the copy, got none")
	}
}
