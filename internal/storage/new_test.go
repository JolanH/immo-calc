package storage

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateNewWritesAnEmptyDocumentAndSwitches(t *testing.T) {
	s, current := loadedStore(t, currentContent)
	s.now = clockAt(fixedNow)
	target := filepath.Join(t.TempDir(), "nouveau.yaml")

	if err := s.CreateNew(target); err != nil {
		t.Fatalf("create: %v", err)
	}

	if s.Path() != target {
		t.Errorf("path = %s", s.Path())
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 0 {
		t.Errorf("items = %+v, want empty", items)
	}
	content, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(content), "schema_version: 1") {
		t.Errorf("new file = %q, err = %v", content, err)
	}
	if names := dirEntries(t, filepath.Dir(current)); len(names) != 2 {
		t.Errorf("files = %v, want a backup of the previous file", names)
	}
	// The new file is immediately the save target.
	if err := Update(s, "items", addItem("a", 1)); err != nil {
		t.Fatalf("save to the new file: %v", err)
	}
}

func TestCreateNewNeverOverwritesAnExistingFile(t *testing.T) {
	s, current := loadedStore(t, currentContent)
	existing := writeFile(t, "precious data, not even YAML\n")

	err := s.CreateNew(existing)

	if !errors.Is(err, os.ErrExist) || !strings.Contains(err.Error(), "existe déjà") {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile(existing); string(got) != "precious data, not even YAML\n" {
		t.Error("the existing file was modified")
	}
	if s.Path() != current {
		t.Errorf("path = %s, want unchanged", s.Path())
	}
	if names := dirEntries(t, filepath.Dir(current)); len(names) != 1 {
		t.Errorf("files = %v, want no backup", names)
	}
}

func TestCreateNewInAMissingFolderFails(t *testing.T) {
	s, current := loadedStore(t, currentContent)

	err := s.CreateNew(filepath.Join(t.TempDir(), "no-such-dir", "x.yaml"))

	if err == nil || !strings.Contains(err.Error(), "Impossible de créer") {
		t.Fatalf("err = %v", err)
	}
	if s.Path() != current {
		t.Errorf("path = %s", s.Path())
	}
}

func TestNewEndpoint(t *testing.T) {
	f := newOpenFixture(t)
	f.picker.path = filepath.Join(t.TempDir(), "famille")

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/new", localOrigin, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	want := f.picker.path + ".yaml" // the extension is added when missing
	var body infoBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Path != want || body.Counts["items"] != 0 {
		t.Errorf("body = %+v", body)
	}
	if cfg, _ := ReadAppConfig(f.configPath); cfg.LastDataFile != want {
		t.Errorf("remembered = %q", cfg.LastDataFile)
	}
}

func TestNewEndpointRefusesAnExistingFile(t *testing.T) {
	f := newOpenFixture(t)
	existing := writeFile(t, "schema_version: 1\n")
	body, _ := json.Marshal(map[string]string{"path": existing})

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/new", localOrigin, string(body)))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"file_exists"`) ||
		!strings.Contains(rec.Body.String(), "utilisez « Ouvrir »") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if f.store.Path() != f.current {
		t.Error("switched")
	}
}

func TestNewEndpointCancelAndUnavailable(t *testing.T) {
	for _, tt := range []struct {
		err  error
		code int
	}{
		{ErrPickerCancelled, http.StatusNoContent},
		{ErrPickerUnavailable, http.StatusBadRequest},
	} {
		f := newOpenFixture(t)
		f.picker.err = tt.err

		rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/new", localOrigin, ""))

		if rec.Code != tt.code {
			t.Errorf("%v: status = %d, want %d", tt.err, rec.Code, tt.code)
		}
		if f.store.Path() != f.current {
			t.Errorf("%v: switched", tt.err)
		}
	}
}
