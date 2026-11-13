// Package storage keeps all of immo-calc's data in one versioned YAML file.
// It knows nothing about loans or properties: each module registers its own
// top-level section and validator, then reads and writes it with Get/Update.
package storage

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"go.yaml.in/yaml/v3"
)

const versionKey = "schema_version"

// Validator checks a section's raw YAML. It returns a *FieldError to point at
// the offending value (so the error can name its line), or any other error.
type Validator func(raw []byte) error

// Store is the in-memory document, backed by one data file.
type Store struct {
	mu         sync.Mutex
	path       string
	exists     bool   // the data file exists, as far as we know
	hash       []byte // SHA-256 of the file as last read or written
	migrated   bool   // loaded from an older schema_version, not saved since
	fs         fileSystem
	now        func() time.Time
	sections   map[string]*yaml.Node // section key → value node
	validators map[string]Validator
	order      []string // registration order, used when writing
	migrations map[int]Migration
	docChecks  []DocumentCheck
}

// New returns an empty store. Register sections, then call Load.
func New() *Store {
	return &Store{
		sections:   map[string]*yaml.Node{},
		validators: map[string]Validator{},
		migrations: migrations,
		fs:         osFS{},
		now:        time.Now,
	}
}

// RegisterSection declares a top-level section and its validator. It must be
// called before Load.
func (s *Store) RegisterSection(key string, validate Validator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.validators[key]; !dup {
		s.order = append(s.order, key)
	}
	s.validators[key] = validate
}

// Path is the active data file.
func (s *Store) Path() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path
}

// FileExists reports whether the data file existed when it was loaded.
func (s *Store) FileExists() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exists
}

// Migrated reports whether the file was upgraded from an older schema version.
func (s *Store) Migrated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.migrated
}

// Load reads, migrates and validates the data file at path, and makes it the
// active document. A missing file gives an empty document (created on the
// first save). On error the store is left unchanged.
func (s *Store) Load(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.fs.ReadFile(path)
	exists := true
	if errors.Is(err, os.ErrNotExist) {
		data, exists = nil, false
	} else if err != nil {
		return &LoadError{File: path, Message: "lecture impossible (" + err.Error() + ")"}
	}

	sections, migrated, err := s.parse(path, data)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	s.path, s.exists, s.migrated, s.sections, s.hash = path, exists, migrated, sections, sum[:]
	return nil
}

// parse turns file content into validated sections at CurrentVersion.
func (s *Store) parse(file string, data []byte) (map[string]*yaml.Node, bool, error) {
	sections := map[string]*yaml.Node{}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, false, syntaxError(file, err)
	}
	if doc.Kind == 0 { // empty file
		return sections, false, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, false, &LoadError{File: file, Line: root.Line,
			Message: "structure inattendue, le fichier doit contenir des sections « clé: valeur »"}
	}

	version, versionLine := -1, 0
	keyLines := map[string]int{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		k, v := root.Content[i], root.Content[i+1]
		if _, dup := keyLines[k.Value]; dup {
			return nil, false, &LoadError{File: file, Line: k.Line,
				Message: fmt.Sprintf("la section « %s » apparaît deux fois", k.Value)}
		}
		keyLines[k.Value] = k.Line
		if k.Value == versionKey {
			n, err := strconv.Atoi(v.Value)
			if err != nil || v.Kind != yaml.ScalarNode {
				return nil, false, &LoadError{File: file, Line: v.Line,
					Message: "schema_version doit être un nombre entier"}
			}
			version, versionLine = n, v.Line
			continue
		}
		sections[k.Value] = v
	}

	if version < 0 {
		return nil, false, &LoadError{File: file, Message: "schema_version manquant"}
	}
	if version > CurrentVersion {
		return nil, false, &LoadError{File: file, Line: versionLine, Message: fmt.Sprintf(
			"version de fichier %d trop récente, cette version d'immo-calc lit au plus la version %d ; mettez immo-calc à jour",
			version, CurrentVersion)}
	}
	for v := version; v < CurrentVersion; v++ {
		migrate, ok := s.migrations[v]
		if !ok {
			return nil, false, &LoadError{File: file, Line: versionLine,
				Message: fmt.Sprintf("aucune migration depuis la version %d", v)}
		}
		if err := migrate(sections); err != nil {
			return nil, false, &LoadError{File: file,
				Message: fmt.Sprintf("migration depuis la version %d impossible (%v)", v, err)}
		}
	}

	for key, node := range sections {
		validate, known := s.validators[key]
		if !known {
			return nil, false, &LoadError{File: file, Line: keyLines[key],
				Message: fmt.Sprintf("section inconnue « %s »", key)}
		}
		if err := validateNode(key, node, validate); err != nil {
			return nil, false, &LoadError{File: file, Line: err.line, Message: err.message}
		}
	}
	if err := s.checkDocument(sections); err != nil {
		return nil, false, documentLoadError(file, sections, err)
	}
	return sections, version < CurrentVersion, nil
}

type sectionError struct {
	line    int
	message string
}

// validateNode runs a section validator and locates its error in the file.
func validateNode(key string, node *yaml.Node, validate Validator) *sectionError {
	raw, err := yaml.Marshal(node)
	if err != nil {
		return &sectionError{node.Line, fmt.Sprintf("section « %s » illisible (%v)", key, err)}
	}
	err = validate(raw)
	if err == nil {
		return nil
	}
	var fieldErr *FieldError
	if errors.As(err, &fieldErr) {
		return &sectionError{lineOf(node, fieldErr.Path), fieldErr.Message}
	}
	return &sectionError{node.Line, fmt.Sprintf("section « %s » invalide (%v)", key, err)}
}

// Get decodes a section into a T. A missing section gives T's zero value.
func Get[T any](s *Store, key string) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out T
	if _, known := s.validators[key]; !known {
		return out, fmt.Errorf("get section %q: not registered", key)
	}
	node := s.sections[key]
	if node == nil {
		return out, nil
	}
	if err := node.Decode(&out); err != nil {
		return out, fmt.Errorf("decode section %q: %w", key, err)
	}
	return out, nil
}

// Reload re-reads the active data file from disk, e.g. after an external
// edit. If the file is invalid, the in-memory document is kept.
func (s *Store) Reload() error {
	return s.Load(s.Path())
}
