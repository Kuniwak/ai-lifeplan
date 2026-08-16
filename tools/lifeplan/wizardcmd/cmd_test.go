package wizardcmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kuniwak/lifeplan/cli"
	"github.com/Kuniwak/lifeplan/tools/lifeplan/wizardcmd"
)

const theAnswers = "1\n1\n1\n1\n1\n1\n1\n1\n1\n1\n"

func TestTheWizardShouldRefuseToWriteWithoutAName(t *testing.T) {
	spy := cli.SpyProcInout(theAnswers)

	if status := wizardcmd.NewCommandFunc()([]string{"-root", "../../.."}, spy.NewProcInout()); status == 0 {
		t.Error("-name 無しで通った。既定の名前は既定の project を上書きする")
	}
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()

	err := filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		switch {
		case rel == ".git" || rel == "out" || rel == "bin" || rel == ".wt":
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		case rel == ".":
			return nil
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(to, rel), 0o777)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), body, 0o666)
	})
	if err != nil {
		t.Fatalf("copying the repository: %v", err)
	}
}
