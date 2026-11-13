package storage

import (
	"crypto/sha256"
	"errors"
	"os"
)

// missingFileError is a chosen data file that doesn't exist.
type missingFileError struct{ path string }

func (e *missingFileError) Error() string { return "Le fichier " + e.path + " est introuvable" }
func (e *missingFileError) Unwrap() error { return os.ErrNotExist }

// SwitchTo makes the existing data file at path the active one. It is loaded
// and validated first; if that fails nothing changes. Otherwise the current
// data file is backed up before switching.
func (s *Store) SwitchTo(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.fs.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &missingFileError{path: path}
	}
	if err != nil {
		return &LoadError{File: path, Message: "lecture impossible (" + err.Error() + ")"}
	}
	sections, migrated, err := s.parse(path, data)
	if err != nil {
		return err
	}
	if err := s.backupLocked(); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	s.path, s.exists, s.migrated, s.sections, s.hash = path, true, migrated, sections, sum[:]
	return nil
}
