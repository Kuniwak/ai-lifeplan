package law

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/digesttest"
	"github.com/Kuniwak/lifeplan/tsv"
)

const theOriginalsRoot = "../testdata/law"

var theRowsWithoutAnOriginal = map[string]string{
	NationalPensionPremiumTableName: "2005〜2022 年度の行。厚生労働省の発表は各年度 2 年ぶんしか載せず、" +
		"令和5年度より前に届かない。**計画のどの年にも当たらない**——妻が第1号被保険者になるのは 2031 年から",
}

func TestEveryLawRowShouldNameAnOriginalInTheRepository(t *testing.T) {
	tables, err := filepath.Glob(filepath.Join("..", LawDirectory, "*", "*.tsv"))
	if err != nil {
		t.Fatalf("filepath.Glob: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("data/law に表が 1 つも無い")
	}

	for _, path := range tables {
		name := strings.TrimSuffix(
			filepath.ToSlash(strings.TrimPrefix(path, filepath.Join("..", LawDirectory)+string(filepath.Separator))), ".tsv")

		t.Run(name, func(t *testing.T) {
			written, err := tsv.ReadFile(path)
			if err != nil {
				t.Fatalf("tsv.ReadFile: %v", err)
			}
			read, err := tsv.NewReader(written, tsv.Slot(name), LawSourceColumn)
			if err != nil {
				t.Fatalf("tsv.NewReader: %v", err)
			}

			for row := range read.Rows() {
				source := read.Field(row, LawSourceColumn)
				named := theOriginalsNamedIn(source)

				if len(named) == 0 {
					if reason, allowed := theRowsWithoutAnOriginal[name]; allowed {
						t.Logf("行 %d は原本がまだ無い: %s", row+1, reason)
						continue
					}
					t.Errorf("行 %d の 出典 が repo の中の原本を指していない。**URL は出典ではない**——"+
						"死んだり、値の載っていない条を指したり、本文の入っていない頁だったりする。\n  %s",
						row+1, source)
					continue
				}

				for _, path := range named {
					if _, err := os.Stat(path); err != nil {
						t.Errorf("行 %d が %s を指しているが、それは無い: %v", row+1, path, err)
					}
				}
			}
		})
	}
}

func theOriginalsNamedIn(source string) []string {
	var named []string
	for _, field := range strings.FieldsFunc(source, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '、' || r == '。' || r == '（' || r == '）' ||
			r == '・' || r == '「' || r == '」' || r == '*'
	}) {
		index := strings.Index(field, "testdata/law/")
		if index < 0 {
			continue
		}
		named = append(named, filepath.Join("..", strings.TrimSuffix(field[index:], "/")))
	}
	return named
}

func TestEveryOriginalUnderTestdataLawShouldMatchItsDigest(t *testing.T) {
	digesttest.Check(t, filepath.Join(theOriginalsRoot, "originals.tsv"), []string{theOriginalsRoot},
		func(path string) bool { return filepath.Ext(path) == ".md" })
}
