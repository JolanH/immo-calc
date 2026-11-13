package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// item is a test section type standing in for loans/properties.
type item struct {
	Name string `yaml:"name"`
	Qty  int    `yaml:"qty"`
}

func validateItems(raw []byte) error {
	var items []item
	if err := yaml.Unmarshal(raw, &items); err != nil {
		return err
	}
	for i, it := range items {
		if it.Qty < 0 {
			return &FieldError{Path: fmt.Sprintf("[%d].qty", i), Message: fmt.Sprintf("quantité négative pour « %s »", it.Name)}
		}
	}
	return nil
}

type settings struct {
	Rate float64 `yaml:"rate"`
}

func newTestStore() *Store {
	s := New()
	s.RegisterSection("items", validateItems)
	s.RegisterSection("settings", func([]byte) error { return nil })
	return s
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadReadsTypedSections(t *testing.T) {
	path := writeFile(t, `schema_version: 1
settings:
  rate: 3.5
items:
  - name: a
    qty: 1
  - name: b
    qty: 2
`)
	s := newTestStore()

	if err := s.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	items, err := Get[[]item](s, "items")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1] != (item{"b", 2}) {
		t.Errorf("items = %+v", items)
	}
	st, err := Get[settings](s, "settings")
	if err != nil || st.Rate != 3.5 {
		t.Errorf("settings = %+v, err = %v", st, err)
	}
	if s.Path() != path || !s.FileExists() {
		t.Errorf("path = %s, exists = %v", s.Path(), s.FileExists())
	}
}

func TestLoadMissingFileGivesAnEmptyDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.yaml")
	s := newTestStore()

	if err := s.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	items, err := Get[[]item](s, "items")
	if err != nil || len(items) != 0 {
		t.Errorf("items = %+v, err = %v", items, err)
	}
	if s.FileExists() {
		t.Error("FileExists = true for a missing file")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Error("loading must not create the file")
	}
}

func TestLoadMissingSectionsAreEmpty(t *testing.T) {
	s := newTestStore()

	if err := s.Load(writeFile(t, "schema_version: 1\n")); err != nil {
		t.Fatalf("load: %v", err)
	}

	st, err := Get[settings](s, "settings")
	if err != nil || st != (settings{}) {
		t.Errorf("settings = %+v, err = %v", st, err)
	}
}

func TestGetUnregisteredSectionFails(t *testing.T) {
	s := newTestStore()
	_ = s.Load(writeFile(t, "schema_version: 1\n"))

	if _, err := Get[settings](s, "nope"); err == nil {
		t.Error("expected an error")
	}
}

func TestLoadErrorsNameTheFileAndLine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLine string
		wantText string
	}{
		{
			name:     "malformed yaml",
			content:  "schema_version: 1\nitems:\n  - name: a\n   qty: [\n",
			wantText: "YAML",
		},
		{
			name:     "unknown top-level key",
			content:  "schema_version: 1\nitems: []\nitmes: []\n",
			wantLine: "(ligne 3)",
			wantText: "« itmes »",
		},
		{
			name:     "duplicate top-level key",
			content:  "schema_version: 1\nitems: []\nitems: []\n",
			wantLine: "(ligne 3)",
			wantText: "« items »",
		},
		{
			name:     "invalid section content",
			content:  "schema_version: 1\nitems:\n  - name: a\n    qty: 1\n  - name: b\n    qty: -3\n",
			wantLine: "(ligne 6)",
			wantText: "quantité négative pour « b »",
		},
		{
			name:     "section of the wrong type",
			content:  "schema_version: 1\nitems: hello\n",
			wantLine: "(ligne 2)",
			wantText: "items",
		},
		{
			name:     "too new schema version",
			content:  "schema_version: 2\n",
			wantLine: "(ligne 1)",
			wantText: "version",
		},
		{
			name:     "missing schema version",
			content:  "items: []\n",
			wantText: "schema_version",
		},
		{
			name:     "schema version not a number",
			content:  "schema_version: abc\n",
			wantLine: "(ligne 1)",
			wantText: "schema_version",
		},
		{
			name:     "root is not a mapping",
			content:  "- a\n- b\n",
			wantText: "structure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, tt.content)
			s := newTestStore()

			err := s.Load(path)

			if err == nil {
				t.Fatal("expected an error")
			}
			var loadErr *LoadError
			if !errors.As(err, &loadErr) {
				t.Fatalf("err = %T %v, want *LoadError", err, err)
			}
			msg := err.Error()
			if !strings.Contains(msg, path) {
				t.Errorf("message %q does not name the file", msg)
			}
			if tt.wantLine != "" && !strings.Contains(msg, tt.wantLine) {
				t.Errorf("message %q does not contain %q", msg, tt.wantLine)
			}
			if !strings.Contains(msg, tt.wantText) {
				t.Errorf("message %q does not contain %q", msg, tt.wantText)
			}
		})
	}
}

func TestMalformedYAMLReportsItsLine(t *testing.T) {
	s := newTestStore()

	err := s.Load(writeFile(t, "schema_version: 1\nitems:\n  - name: a\n    qty: 1\n  bad: [\n"))

	var loadErr *LoadError
	if !errors.As(err, &loadErr) || loadErr.Line == 0 {
		t.Fatalf("err = %v, want a LoadError with a line", err)
	}
}

func TestEmptyFileIsAnEmptyDocument(t *testing.T) {
	s := newTestStore()

	if err := s.Load(writeFile(t, "")); err != nil {
		t.Fatalf("load: %v", err)
	}
}

func TestOlderFileIsMigratedInMemory(t *testing.T) {
	path := writeFile(t, "schema_version: 0\nthings:\n  - name: a\n    qty: 4\n")
	original, _ := os.ReadFile(path)
	s := newTestStore()
	// Fake v0 → v1 migration: the "things" section was renamed "items".
	s.migrations = map[int]Migration{
		0: func(sections map[string]*yaml.Node) error {
			sections["items"] = sections["things"]
			delete(sections, "things")
			return nil
		},
	}

	if err := s.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}

	items, _ := Get[[]item](s, "items")
	if len(items) != 1 || items[0] != (item{"a", 4}) {
		t.Errorf("items = %+v", items)
	}
	if !s.Migrated() {
		t.Error("Migrated() = false")
	}
	if now, _ := os.ReadFile(path); string(now) != string(original) {
		t.Error("loading must not write the migrated document")
	}
}

func TestMissingMigrationIsAnError(t *testing.T) {
	s := newTestStore()
	s.migrations = map[int]Migration{}

	if err := s.Load(writeFile(t, "schema_version: 0\n")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidatedSectionsOnlyRunForPresentSections(t *testing.T) {
	called := false
	s := New()
	s.RegisterSection("items", func([]byte) error { called = true; return nil })

	if err := s.Load(writeFile(t, "schema_version: 1\n")); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("validator called for a missing section")
	}
}
