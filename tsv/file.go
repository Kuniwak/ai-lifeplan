package tsv

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

const FileMode os.FileMode = 0o644

func ReadFile(path string) (*Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("tsv.ReadFile: %w", err)
	}
	defer f.Close()

	t, err := Read(f)
	if err != nil {
		return nil, fmt.Errorf("tsv.ReadFile: %s: %w", path, err)
	}
	return t, nil
}

func WriteFile(path string, t *Table) error {
	var buf bytes.Buffer
	if err := Write(&buf, t); err != nil {
		return fmt.Errorf("tsv.WriteFile: %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("tsv.WriteFile: %s: %w", path, err)
	}
	tmpPath := tmp.Name()

	if err := writeAndClose(tmp, buf.Bytes()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("tsv.WriteFile: %s: %w", path, err)
	}

	if err := os.Chmod(tmpPath, FileMode); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("tsv.WriteFile: %s: %w", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("tsv.WriteFile: %s: %w", path, err)
	}
	return nil
}

func writeAndClose(f *os.File, b []byte) error {
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func Under(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.FromSlash(path)
	}
	return filepath.Join(root, filepath.FromSlash(path))
}
