package sheets

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

const Extension = ".tsv"

type Copy struct {
	fsys fs.FS
}

func New(fsys fs.FS) Copy {
	return Copy{fsys: fsys}
}

func (c Copy) Names() ([]string, error) {
	entries, err := fs.ReadDir(c.fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("sheets.Names: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), Extension) {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), Extension))
	}
	slices.Sort(names)
	return names, nil
}

func (c Copy) Table(name string) (*tsv.Table, error) {
	f, err := c.fsys.Open(path.Join(name + Extension))
	if err != nil {
		return nil, fmt.Errorf("sheets.Table: %s: %w", name, err)
	}
	defer f.Close()

	table, err := tsv.Read(f)
	if err != nil {
		return nil, fmt.Errorf("sheets.Table: %s: %w", name, err)
	}
	return table, nil
}
