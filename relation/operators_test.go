package relation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var operators = []string{
	"Aggregate",
	"Join",
	"Lag",
	"LeftJoin",
	"Lookup",
	"Map",

	"MapEach",
}

var constructorsAndAccessors = []string{
	"At", "Constant", "Len", "Max", "Min", "New", "NewBands", "Over",
	"Rows", "SameEveryYear", "Span", "Years",
}

var periods = []string{
	"All", "BandsOfPeriods", "Bounds", "Covers", "From", "Key", "NewPeriod", "NewPeriods", "Overlap", "String",
	"To", "Unbounded", "Value",
}

var calendar = []string{"MonthsSince", "YearMonthOf"}

func TestThePackageShouldOfferExactlyTheAgreedOperators(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var got []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !fn.Name.IsExported() {
					continue
				}
				got = append(got, fn.Name.Name)
			}
		}
	}
	slices.Sort(got)
	got = slices.Compact(got)

	want := slices.Concat(operators, constructorsAndAccessors, calendar, periods)
	slices.Sort(want)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("the exported surface of relation changed (-want +got):\n%s\n"+
			"Adding an operator is a decision, not a detail.", diff)
	}
}
