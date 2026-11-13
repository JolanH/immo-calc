package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v3"
)

func loadedStore(t *testing.T, content string) (*Store, string) {
	t.Helper()
	path := writeFile(t, content)
	s := newTestStore()
	if err := s.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	return s, path
}

func addItem(name string, qty int) func(*[]item) error {
	return func(items *[]item) error {
		*items = append(*items, item{name, qty})
		return nil
	}
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestUpdateSavesAndSurvivesAReload(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nitems:\n  - name: a\n    qty: 1\n")

	if err := Update(s, "items", addItem("b", 2)); err != nil {
		t.Fatalf("update: %v", err)
	}

	reloaded := newTestStore()
	if err := reloaded.Load(path); err != nil {
		t.Fatalf("reload: %v", err)
	}
	items, _ := Get[[]item](reloaded, "items")
	if len(items) != 2 || items[1] != (item{"b", 2}) {
		t.Errorf("items after reload = %+v", items)
	}
	content, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(content), "# Fichier de données immo-calc — ne pas modifier schema_version\nschema_version: 1\n") {
		t.Errorf("file content:\n%s", content)
	}
}

func TestFirstUpdateCreatesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	s := newTestStore()
	if err := s.Load(path); err != nil {
		t.Fatal(err)
	}

	if err := Update(s, "items", addItem("a", 1)); err != nil {
		t.Fatalf("update: %v", err)
	}

	if !s.FileExists() {
		t.Error("FileExists = false after the first save")
	}
	reloaded := newTestStore()
	if err := reloaded.Load(path); err != nil {
		t.Fatalf("reload: %v", err)
	}
	items, _ := Get[[]item](reloaded, "items")
	if len(items) != 1 {
		t.Errorf("items = %+v", items)
	}
}

func TestUpdateKeepsOtherSections(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nsettings:\n  rate: 2.5\n")

	if err := Update(s, "items", addItem("a", 1)); err != nil {
		t.Fatal(err)
	}

	reloaded := newTestStore()
	_ = reloaded.Load(path)
	st, _ := Get[settings](reloaded, "settings")
	if st.Rate != 2.5 {
		t.Errorf("settings lost: %+v", st)
	}
}

func TestUpdateCallbackErrorSavesNothing(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nitems: []\n")
	before, _ := os.ReadFile(path)
	boom := errors.New("boom")

	err := Update(s, "items", func(items *[]item) error {
		*items = append(*items, item{"a", 1})
		return boom
	})

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("file changed")
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 0 {
		t.Errorf("memory changed: %+v", items)
	}
}

func TestUpdateRejectsInvalidSection(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nitems: []\n")
	before, _ := os.ReadFile(path)

	err := Update(s, "items", addItem("bad", -1))

	var fieldErr *FieldError
	if !errors.As(err, &fieldErr) || fieldErr.Path != "[0].qty" {
		t.Fatalf("err = %v, want a FieldError on [0].qty", err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("file changed")
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 0 {
		t.Errorf("memory changed: %+v", items)
	}
}

func TestUpdateUnregisteredSectionFails(t *testing.T) {
	s, _ := loadedStore(t, "schema_version: 1\n")

	if err := Update(s, "nope", func(*settings) error { return nil }); err == nil {
		t.Error("expected an error")
	}
}

// failingFS makes one step of the atomic write fail.
type failingFS struct {
	osFS
	failWrite, failRename bool
}

func (f failingFS) WriteTemp(dir, pattern string, data []byte) (string, error) {
	if f.failWrite {
		name, err := f.osFS.WriteTemp(dir, pattern, data[:len(data)/2])
		if err == nil {
			_ = os.Remove(name)
		}
		return "", errors.New("disk full")
	}
	return f.osFS.WriteTemp(dir, pattern, data)
}

func (f failingFS) Rename(oldPath, newPath string) error {
	if f.failRename {
		return errors.New("rename failed")
	}
	return f.osFS.Rename(oldPath, newPath)
}

func TestFailedWriteLeavesTheOriginalFileUntouched(t *testing.T) {
	for _, tt := range []struct {
		name string
		fs   failingFS
	}{
		{"write fails", failingFS{failWrite: true}},
		{"rename fails", failingFS{failRename: true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, path := loadedStore(t, "schema_version: 1\nitems:\n  - name: a\n    qty: 1\n")
			before, _ := os.ReadFile(path)
			s.fs = tt.fs

			if err := Update(s, "items", addItem("b", 2)); err == nil {
				t.Fatal("expected an error")
			}

			if after, _ := os.ReadFile(path); string(after) != string(before) {
				t.Errorf("file changed:\n%s", after)
			}
			if names := dirEntries(t, filepath.Dir(path)); len(names) != 1 {
				t.Errorf("leftover files: %v", names)
			}
			if items, _ := Get[[]item](s, "items"); len(items) != 1 {
				t.Errorf("memory changed: %+v", items)
			}
		})
	}
}

func TestExternalEditIsDetected(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nitems: []\n")
	external := "schema_version: 1\nitems:\n  - name: edited\n    qty: 9\n"
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Update(s, "items", addItem("a", 1))

	if !errors.Is(err, ErrFileChangedExternally) {
		t.Fatalf("err = %v, want ErrFileChangedExternally", err)
	}
	if now, _ := os.ReadFile(path); string(now) != external {
		t.Error("the external edit was overwritten")
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 0 {
		t.Errorf("memory changed: %+v", items)
	}
}

func TestFileCreatedExternallyAfterLoadIsAConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	s := newTestStore()
	_ = s.Load(path)
	if err := os.WriteFile(path, []byte("schema_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Update(s, "items", addItem("a", 1)); !errors.Is(err, ErrFileChangedExternally) {
		t.Fatalf("err = %v, want ErrFileChangedExternally", err)
	}
}

func TestFileDeletedExternallyAfterLoadIsAConflict(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\n")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	if err := Update(s, "items", addItem("a", 1)); !errors.Is(err, ErrFileChangedExternally) {
		t.Fatalf("err = %v, want ErrFileChangedExternally", err)
	}
}

func TestSuccessiveSavesDoNotConflictAndMakeNoBackup(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\n")

	for i := range 3 {
		if err := Update(s, "items", addItem("x", i)); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	if names := dirEntries(t, filepath.Dir(path)); len(names) != 1 {
		t.Errorf("files in the data dir: %v, want only the data file", names)
	}
}

func TestMigratedDocumentIsWrittenAtCurrentVersionOnNextSave(t *testing.T) {
	path := writeFile(t, "schema_version: 0\n")
	s := newTestStore()
	s.migrations = map[int]Migration{0: func(map[string]*yaml.Node) error { return nil }}
	if err := s.Load(path); err != nil {
		t.Fatal(err)
	}

	if err := Update(s, "items", addItem("a", 1)); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "schema_version: 1\n") {
		t.Errorf("file not upgraded:\n%s", content)
	}
	if s.Migrated() {
		t.Error("Migrated() still true after saving")
	}
}

func TestConcurrentUpdatesAreSerialised(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\n")
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if err := Update(s, "items", addItem("x", 1)); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	reloaded := newTestStore()
	_ = reloaded.Load(path)
	if items, _ := Get[[]item](reloaded, "items"); len(items) != 20 {
		t.Errorf("items = %d, want 20", len(items))
	}
}
