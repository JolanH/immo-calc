package storage

import (
	"errors"

	"go.yaml.in/yaml/v3"
)

// DocumentCheck validates rules that span sections (e.g. a property's loan
// must exist). get returns a section's raw YAML, or nil when it's missing.
type DocumentCheck func(get func(key string) []byte) error

// RegisterDocumentCheck adds a check run after the section validators on
// load, import, open and reload, and before every save.
func (s *Store) RegisterDocumentCheck(check DocumentCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docChecks = append(s.docChecks, check)
}

// checkDocument runs the document checks on sections.
func (s *Store) checkDocument(sections map[string]*yaml.Node) error {
	get := func(key string) []byte {
		node := sections[key]
		if node == nil {
			return nil
		}
		raw, err := yaml.Marshal(node)
		if err != nil {
			return nil
		}
		return raw
	}
	for _, check := range s.docChecks {
		if err := check(get); err != nil {
			return err
		}
	}
	return nil
}

// documentLoadError turns a document check failure into a LoadError,
// locating its line when the check pointed at a value.
func documentLoadError(file string, sections map[string]*yaml.Node, err error) *LoadError {
	var docErr *DocumentError
	if errors.As(err, &docErr) {
		line := 0
		if node := sections[docErr.Section]; node != nil {
			line = lineOf(node, docErr.Path)
		}
		return &LoadError{File: file, Line: line, Message: docErr.Message}
	}
	return &LoadError{File: file, Message: err.Error()}
}
