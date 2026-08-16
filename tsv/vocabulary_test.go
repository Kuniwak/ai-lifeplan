package tsv_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestEveryConstantInATypedBlockShouldCarryTheType(t *testing.T) {
	var untyped []string
	blocks := 0
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir():
			if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".." {
				return filepath.SkipDir
			}
			return nil
		case !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go"):
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST || !mentionsAStringType(gen) {
				continue
			}
			blocks++
			for _, spec := range gen.Specs {
				value := spec.(*ast.ValueSpec)
				if value.Type != nil || !isStringLiteral(value) {
					continue
				}
				for _, name := range value.Names {
					untyped = append(untyped, path+": "+name.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.WalkDir: %v", err)
	}

	if blocks < 20 {
		t.Fatalf("型付きの定数ブロックが %d 個しか見つからない。この検査が空回りしている", blocks)
	}
	for _, name := range untyped {
		t.Errorf("%s に型が無い。型付きの定数と同じブロックに並んでいるが、"+
			"型無しの文字列定数はどの string 型にも通ってしまう", name)
	}
}

func mentionsAStringType(gen *ast.GenDecl) bool {
	for _, spec := range gen.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok || value.Type == nil {
			continue
		}
		switch value.Type.(type) {
		case *ast.SelectorExpr, *ast.Ident:
			return true
		}
	}
	return false
}

func isStringLiteral(value *ast.ValueSpec) bool {
	for _, expr := range value.Values {
		if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			return true
		}
	}
	return false
}
