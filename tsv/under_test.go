package tsv_test

import (
	"path/filepath"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestUnder(t *testing.T) {
	for _, test := range []struct {
		name string
		root string
		path string
		want string
	}{
		{name: "相対は root から取る", root: "/repo", path: "data/x.tsv", want: "/repo/data/x.tsv"},
		{name: "絶対はそのまま", root: "/repo", path: "/dev/fd/63", want: "/dev/fd/63"},
		{name: "root が . でも絶対はそのまま", root: ".", path: "/dev/fd/63", want: "/dev/fd/63"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := tsv.Under(test.root, test.path); got != filepath.FromSlash(test.want) {
				t.Errorf("Under(%q, %q) = %q, want %q", test.root, test.path, got, test.want)
			}
		})
	}
}
