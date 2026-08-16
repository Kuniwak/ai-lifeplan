package law

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const theEgovDir = "../testdata/law/egov"

var (
	theTagPattern   = regexp.MustCompile(`<[^>]*>`)
	theSpacePattern = regexp.MustCompile(`\s+`)
	theKanjiNumeral = `[一二三四五六七八九十百千万億]+`
)

func egovArticle(t *testing.T, name string) string {
	t.Helper()

	body, err := os.ReadFile(filepath.Join(theEgovDir, name))
	if err != nil {
		t.Fatalf("egovArticle: %v", err)
	}
	flat := theSpacePattern.ReplaceAllString(theTagPattern.ReplaceAllString(string(body), " "), " ")

	if !strings.Contains(flat, "第") {
		t.Fatalf("egovArticle: %s に条文が入っていない", name)
	}
	return flat
}

func egovAmount(t *testing.T, article, pattern string) int64 {
	t.Helper()

	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(article, -1)
	switch len(matches) {
	case 1:
	case 0:
		t.Fatalf("egovAmount: 条文に %s が無い。**条文が変わったのか、読み方が違うのか**を確かめること", pattern)
	default:
		t.Fatalf("egovAmount: 条文の %d 箇所が %s に当たる。どの額を指しているか決まらない", len(matches), pattern)
	}

	amount, err := ParseKanjiNumber(matches[0][1])
	if err != nil {
		t.Fatalf("egovAmount: %v", err)
	}
	return amount
}
