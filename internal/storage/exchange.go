package storage

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// importLabel names an imported document in LoadError messages:
// « Le fichier importé est invalide… ».
const importLabel = "importé"

// Export returns the in-memory document in the data file format.
func (s *Store) Export() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encode(s.sections)
}

// Import replaces all data with the given document. It is parsed, migrated
// and validated completely first; if that fails, nothing changes. Otherwise
// the current data file is backed up and overwritten. The data file path
// stays the same.
func (s *Store) Import(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sections, _, err := s.parse(importLabel, data)
	if err != nil {
		return err
	}
	return s.replaceLocked(sections)
}

// replaceLocked backs up the current data file, then writes sections over it.
// Unlike a normal save there is no external-edit check: whatever is on disk
// is preserved in the backup. s.mu must be held.
func (s *Store) replaceLocked(sections map[string]*yaml.Node) error {
	data, err := s.encode(sections)
	if err != nil {
		return err
	}
	if err := s.backupLocked(); err != nil {
		return err
	}
	if err := writeAtomic(s.fs, s.path, data); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	s.sections, s.hash, s.exists, s.migrated = sections, sum[:], true, false
	return nil
}

// backupLocked copies the current data file to
// <name>.bak-YYYYMMDD-HHMMSS.yaml in the same folder. A missing file needs no
// backup. Existing backups are never overwritten.
func (s *Store) backupLocked() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s for backup: %w", s.path, err)
	}
	base := strings.TrimSuffix(s.path, filepath.Ext(s.path)) + ".bak-" + s.now().Format("20060102-150405")
	name := base + ".yaml"
	for i := 2; ; i++ {
		err := writeNew(name, data)
		if err == nil {
			return nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("write backup %s: %w", name, err)
		}
		name = fmt.Sprintf("%s-%d.yaml", base, i)
	}
}

// writeNew writes data to a file that must not exist yet, and fsyncs it.
func writeNew(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	serr := f.Sync()
	cerr := f.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

// exportFileName is the download name for an export made at t.
func exportFileName(t time.Time) string {
	return "immo-calc-export-" + t.Format("2006-01-02") + ".yaml"
}
