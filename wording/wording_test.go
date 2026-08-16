package wording_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/wording"
)

type personName string

func TestAKeyShouldBeQuotedWhenItIsANameAndBareWhenItIsANumber(t *testing.T) {
	type testCase struct {
		Key  wording.Key
		Want string
	}

	testCases := map[string]testCase{
		"名前":             {Key: wording.Name("土地"), Want: `"土地"`},
		"名前のついた文字列型":     {Key: wording.Name(personName("夫")), Want: `"夫"`},
		"空の名前（境界値）":      {Key: wording.Name(""), Want: `""`},
		"端に空白のある名前（境界値）": {Key: wording.Name(" 夫"), Want: `" 夫"`},
		"数":           {Key: wording.Number(2024), Want: "2024"},
		"2 つの欄でひとつの鍵": {Key: wording.Pair("証券会社A", "NISA"), Want: "証券会社A / NISA"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := tc.Key.String()

			if got != tc.Want {
				t.Errorf("%q でなく %q になっている", tc.Want, got)
			}
		})
	}
}

func TestTheTwoReadersShouldGetTheirOwnLanguageAndNoMixture(t *testing.T) {
	finding := wording.DuplicateKeyFinding("契約", wording.Name("土地"), "どちらの条件で返すのか決まらない")
	for _, want := range []string{"契約", `"土地"`, "二度書かれており", "どちらの条件で返すのか決まらない"} {
		if !strings.Contains(finding, want) {
			t.Errorf("finding %q が %q を含んでいない", finding, want)
		}
	}
	if strings.Contains(finding, ":") {
		t.Errorf("finding %q に前置きがある", finding)
	}

	err := wording.DuplicateKeyError("stepfn.At", "year", wording.Number(2024), "which row applies")
	for _, want := range []string{"stepfn.At:", "year", "2024", "written twice", "which row applies"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q が %q を含んでいない", err, want)
		}
	}
}

func TestNoClauseShouldBeWrittenOutsideThisPackage(t *testing.T) {
	clauses := []string{
		string(wording.WhichRowReadsTheYear),
		string(wording.WhichRowAppliesInTheYear),
		string(wording.WhichAmountIsTheRecord),
		string(wording.WhichAnswerIsTaken),
		string(wording.WhichRowAnswersForTheKey),
		string(wording.WhichRowDecidesTheValue),
		string(wording.WhichRowReadsTheYearEn),
		string(wording.WhichRowAppliesInTheYearEn),
		string(wording.WhichAmountIsTheRecordEn),
		string(wording.WhichHoldingItCountsAsEn),
		"が二度書かれており",
		"is written twice, so",
	}

	root := ".."
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && (d.Name() == ".git" || d.Name() == "out" || d.Name() == ".wt"):
			return filepath.SkipDir
		case d.IsDir() || !strings.HasSuffix(path, ".go"):
			return nil
		case strings.HasPrefix(filepath.ToSlash(path), "../wording/"):
			return nil
		}

		read, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for line, text := range strings.Split(string(read), "\n") {
			if strings.HasPrefix(strings.TrimSpace(text), "//") {
				continue
			}
			for _, clause := range clauses {
				if strings.Contains(text, clause) {
					t.Errorf("%s:%d が %q を自分で書いている。wording の定数を使うこと", path, line+1, clause)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir: %v", err)
	}
}
