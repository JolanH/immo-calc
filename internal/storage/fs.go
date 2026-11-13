package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// fileSystem is the file access the store needs, so tests can make one step fail.
type fileSystem interface {
	ReadFile(name string) ([]byte, error)
	// WriteTemp writes data to a new temp file in dir, fsyncs and closes it,
	// and returns its name.
	WriteTemp(dir, pattern string, data []byte) (string, error)
	Rename(oldPath, newPath string) error
	Remove(name string) error
}

type osFS struct{}

func (osFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFS) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }
func (osFS) Remove(name string) error             { return os.Remove(name) }

func (osFS) WriteTemp(dir, pattern string, data []byte) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	_, werr := f.Write(data)
	serr := f.Sync()
	cerr := f.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// writeAtomic replaces path with data: write a temp file in the same
// directory, fsync it, then rename it over path. A crash at any point leaves
// either the old file or the new one, never a truncated one.
func writeAtomic(fsys fileSystem, path string, data []byte) error {
	tmp, err := fsys.WriteTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*", data)
	if err != nil {
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	if err := fsys.Rename(tmp, path); err != nil {
		_ = fsys.Remove(tmp)
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
