package storage

import (
	"crypto/sha256"
	"errors"
	"io/fs"
	"os"

	"go.yaml.in/yaml/v3"
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

// existsError is a "new file" target that already exists.
type existsError struct{}

func (existsError) Error() string {
	return "Ce fichier existe déjà, choisissez un autre nom ou utilisez « Ouvrir »"
}
func (existsError) Unwrap() error { return fs.ErrExist }

// createError is a "new file" target that can't be written.
type createError struct{ err error }

func (e *createError) Error() string { return "Impossible de créer le fichier : " + e.err.Error() }
func (e *createError) Unwrap() error { return e.err }

// CreateNew writes an empty document at path, which must not exist yet, and
// makes it the active file after backing up the current one. Existing files
// are never overwritten.
func (s *Store) CreateNew(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	empty := map[string]*yaml.Node{}
	data, err := s.encode(empty)
	if err != nil {
		return err
	}
	if err := writeNew(path, data); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return existsError{}
		}
		return &createError{err: err}
	}
	if err := s.backupLocked(); err != nil {
		_ = os.Remove(path)
		return err
	}
	sum := sha256.Sum256(data)
	s.path, s.exists, s.migrated, s.sections, s.hash = path, true, false, empty, sum[:]
	return nil
}
