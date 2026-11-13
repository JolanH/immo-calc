package storage

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"os"
	"strconv"

	"go.yaml.in/yaml/v3"
)

// ErrFileChangedExternally means the data file was modified (or created, or
// deleted) by something else since immo-calc last read or wrote it. The save
// is refused so that edit isn't overwritten.
var ErrFileChangedExternally = errors.New("data file changed outside immo-calc")

const fileHeader = "Fichier de données immo-calc — ne pas modifier schema_version"

// Update decodes section key into a T, applies fn, validates the result and
// saves the whole document atomically. If fn, validation or the save fails,
// nothing changes, in memory or on disk. Validation failures are returned as
// the validator's error (often a *FieldError).
func Update[T any](s *Store, key string, fn func(*T) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	validate, known := s.validators[key]
	if !known {
		return fmt.Errorf("update section %q: not registered", key)
	}
	var value T
	if node := s.sections[key]; node != nil {
		if err := node.Decode(&value); err != nil {
			return fmt.Errorf("decode section %q: %w", key, err)
		}
	}
	if err := fn(&value); err != nil {
		return err
	}

	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return fmt.Errorf("encode section %q: %w", key, err)
	}
	raw, err := yaml.Marshal(&node)
	if err != nil {
		return fmt.Errorf("encode section %q: %w", key, err)
	}
	if err := validate(raw); err != nil {
		return err
	}

	sections := maps.Clone(s.sections)
	sections[key] = &node
	if err := s.checkDocument(sections); err != nil {
		return err
	}
	return s.saveLocked(sections)
}

// saveLocked writes sections to the data file and, on success, makes them the
// in-memory document. s.mu must be held.
func (s *Store) saveLocked(sections map[string]*yaml.Node) error {
	data, err := s.encode(sections)
	if err != nil {
		return err
	}
	if err := s.checkUnchangedLocked(); err != nil {
		return err
	}
	if err := writeAtomic(s.fs, s.path, data); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	s.sections, s.hash, s.exists, s.migrated = sections, sum[:], true, false
	return nil
}

// checkUnchangedLocked compares the file on disk with what was last read or written.
func (s *Store) checkUnchangedLocked() error {
	current, err := s.fs.ReadFile(s.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if s.exists {
			return ErrFileChangedExternally
		}
		return nil
	case err != nil:
		return fmt.Errorf("check %s before saving: %w", s.path, err)
	case !s.exists:
		return ErrFileChangedExternally
	}
	sum := sha256.Sum256(current)
	if !bytes.Equal(sum[:], s.hash) {
		return ErrFileChangedExternally
	}
	return nil
}

// encode renders the document: header comment, schema_version, then the
// present sections in registration order.
func (s *Store) encode(sections map[string]*yaml.Node) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode}
	add := func(key string, value *yaml.Node) {
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
	}
	add(versionKey, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(CurrentVersion)})
	root.Content[0].HeadComment = fileHeader
	for _, key := range s.order {
		if node := sections[key]; node != nil {
			add(key, node)
		}
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encode data file: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode data file: %w", err)
	}
	return buf.Bytes(), nil
}
