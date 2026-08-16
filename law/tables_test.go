package law_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/law"
)

func TestEveryTableUnderDataLawShouldHaveAShape(t *testing.T) {
	described := make(map[string]bool, len(law.Shapes()))
	for _, shape := range law.Shapes() {
		described[shape.Name] = true
	}

	root := "../" + law.LawDirectory
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tsv") {
			return err
		}

		relative := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(path), root+"/"), ".tsv")
		if described[relative] {
			return nil
		}
		if described[filepath.Base(relative)] {
			return nil
		}
		t.Errorf("data/law/%s.tsv has no shape, so nothing is checked about it", relative)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
}

func TestEveryShapeShouldNameATableThatIsThere(t *testing.T) {
	fsys := os.DirFS("../" + law.LawDirectory)

	for _, shape := range law.Shapes() {
		if !shape.Regional {
			if _, err := law.LoadShape(fsys, shape, ""); err != nil {
				t.Errorf("%s: %v", shape.Name, err)
			}
			continue
		}

		regions, err := law.RegionsWith(fsys, shape.Name)
		if err != nil {
			t.Errorf("%s: %v", shape.Name, err)
			continue
		}
		if len(regions) == 0 {
			t.Errorf("%s: no region provides it", shape.Name)
		}
		for _, region := range regions {
			if _, err := law.LoadShape(fsys, shape, region); err != nil {
				t.Errorf("%s/%s: %v", region, shape.Name, err)
			}
		}
	}
}
