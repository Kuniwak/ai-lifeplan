package digesttest

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	FileColumn   tsv.ColumnName = "ファイル"
	BytesColumn  tsv.ColumnName = "バイト数"
	DigestColumn tsv.ColumnName = "sha256"
)

func Check(t *testing.T, manifest string, covers []string, exempt func(path string) bool) {
	t.Helper()

	root := filepath.Dir(manifest)
	listed := map[string]bool{}

	var sameAs []os.FileInfo

	written, err := tsv.ReadFile(manifest)
	if err != nil {
		t.Fatalf("tsv.ReadFile(%s): %v", manifest, err)
	}
	read, err := tsv.NewReader(written, tsv.Slot(filepath.Base(manifest)), FileColumn, BytesColumn, DigestColumn)
	if err != nil {
		t.Fatalf("tsv.NewReader: %v", err)
	}

	for row := range read.Rows() {
		name := read.Field(row, FileColumn)
		path := filepath.Join(root, name)
		listed[path] = true
		if info, err := os.Stat(path); err == nil {
			sameAs = append(sameAs, info)
		}

		t.Run(name, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				t.Fatalf("%s が無い。**%s が名指した原本は、そこに置かれていなければ"+
					"ならない**——表を検算する相手が消えている", path, manifest)
			}
			if err != nil {
				t.Fatalf("os.ReadFile: %v", err)
			}

			size, err := read.Count(row, BytesColumn)
			if err != nil {
				t.Fatalf("バイト数: %v", err)
			}
			if len(content) != size {
				t.Errorf("%s は %d バイトだが、%s は %s と言っている",
					name, len(content), filepath.Base(manifest), strconv.Itoa(size))
			}
			sum := sha256.Sum256(content)
			if got, want := hex.EncodeToString(sum[:]), read.Field(row, DigestColumn); got != want {
				t.Errorf("%s の sha256 が %s で、%s の %s と違う。"+
					"表を書いたときに読んだ原本と同じものではない",
					name, got, filepath.Base(manifest), want)
			}
		})
	}

	for _, dir := range covers {
		err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			if filepath.Base(path) == "PROVENANCE.md" || path == manifest {
				return nil
			}
			if listed[path] || (exempt != nil && exempt(path)) {
				return nil
			}
			if info, err := entry.Info(); err == nil {
				for _, already := range sameAs {
					if os.SameFile(already, info) {
						return nil
					}
				}
			}
			t.Errorf("%s が %s に無い。**置いてあるのに誰も検算していない原本である**", path, manifest)
			return nil
		})
		if err != nil {
			t.Fatalf("filepath.WalkDir(%s): %v", dir, err)
		}
	}
}
