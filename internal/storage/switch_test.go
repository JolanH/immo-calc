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

// fakePicker returns a canned answer and records that it was shown.
type fakePicker struct {
	path  string
	err   error
	shown bool
}

func (p *fakePicker) Open() (string, error) {
	p.shown = true
	return p.path, p.err
}

func (p *fakePicker) SaveAs(string) (string, error) {
	p.shown = true
	return p.path, p.err
}

const currentContent = "schema_version: 1\nitems:\n  - name: current\n    qty: 1\n"

func TestSwitchToAValidFileBacksUpTheCurrentOne(t *testing.T) {
	s, current := loadedStore(t, currentContent)
	s.now = clockAt(fixedNow)
	other := writeFile(t, "schema_version: 1\nitems:\n  - name: other\n    qty: 2\n")

	if err := s.SwitchTo(other); err != nil {
		t.Fatalf("switch: %v", err)
	}

	if s.Path() != other {
		t.Errorf("path = %s", s.Path())
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 1 || items[0].Name != "other" {
		t.Errorf("items = %+v", items)
	}
	backup := filepath.Join(filepath.Dir(current), "immo-calc.bak-20261113-140509.yaml")
	if got, err := os.ReadFile(backup); err != nil || string(got) != currentContent {
		t.Errorf("backup = %q, err = %v", got, err)
	}
	// Saves now go to the new file.
	if err := Update(s, "items", addItem("added", 3)); err != nil {
		t.Fatal(err)
	}
	if now, _ := os.ReadFile(current); string(now) != currentContent {
		t.Error("the previous file was modified after switching")
	}
}

func TestSwitchToAnInvalidFileChangesNothing(t *testing.T) {
	s, current := loadedStore(t, currentContent)
	bad := writeFile(t, "schema_version: 1\nitems:\n  - name: x\n    qty: -5\n")

	err := s.SwitchTo(bad)

	var loadErr *LoadError
	if !errors.As(err, &loadErr) || !strings.Contains(err.Error(), bad) {
		t.Fatalf("err = %v", err)
	}
	if s.Path() != current {
		t.Errorf("path = %s, want unchanged", s.Path())
	}
	if names := dirEntries(t, filepath.Dir(current)); len(names) != 1 {
		t.Errorf("files = %v, want no backup", names)
	}
}

func TestSwitchToAMissingFileIsRefused(t *testing.T) {
	s, current := loadedStore(t, currentContent)

	err := s.SwitchTo(filepath.Join(t.TempDir(), "absent.yaml"))

	if err == nil || !strings.Contains(err.Error(), "introuvable") {
		t.Fatalf("err = %v", err)
	}
	if s.Path() != current {
		t.Errorf("path = %s", s.Path())
	}
}

type openFixture struct {
	store      *Store
	current    string
	configPath string
	picker     *fakePicker
	handlers   *Handlers
}

func newOpenFixture(t *testing.T) openFixture {
	t.Helper()
	s, current := loadedStore(t, currentContent)
	s.now = clockAt(fixedNow)
	picker := &fakePicker{}
	configPath := filepath.Join(t.TempDir(), ConfigFile)
	h := NewHandlers(s, HandlerConfig{
		CountKeys:  []string{"items"},
		Warning:    "Le fichier /old est introuvable",
		ConfigPath: configPath,
		Picker:     picker,
	})
	return openFixture{s, current, configPath, picker, h}
}

const localOrigin = "http://127.0.0.1:8421"

func TestOpenWithTheNativeDialog(t *testing.T) {
	f := newOpenFixture(t)
	f.picker.path = writeFile(t, "schema_version: 1\nitems: []\n")

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/open", localOrigin, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body infoBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Path != f.picker.path || body.Warning != nil {
		t.Errorf("body = %+v (the old warning must be cleared)", body)
	}
	cfg, _ := ReadAppConfig(f.configPath)
	if cfg.LastDataFile != f.picker.path {
		t.Errorf("remembered file = %q", cfg.LastDataFile)
	}
	if names := dirEntries(t, filepath.Dir(f.current)); len(names) != 2 {
		t.Errorf("files = %v, want a backup of the previous file", names)
	}
}

func TestOpenCancelled(t *testing.T) {
	f := newOpenFixture(t)
	f.picker.err = ErrPickerCancelled

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/open", localOrigin, ""))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if f.store.Path() != f.current {
		t.Error("switched on cancel")
	}
	if _, err := os.Stat(f.configPath); !errors.Is(err, os.ErrNotExist) {
		t.Error("config written on cancel")
	}
}

func TestOpenWithoutANativeDialogAsksForAPath(t *testing.T) {
	f := newOpenFixture(t)
	f.picker.err = ErrPickerUnavailable

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/open", localOrigin, ""))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"picker_unavailable"`) {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOpenWithATypedPath(t *testing.T) {
	f := newOpenFixture(t)
	other := writeFile(t, "schema_version: 1\nitems: []\n")
	body, _ := json.Marshal(map[string]string{"path": other})

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/open", localOrigin, string(body)))

	if rec.Code != http.StatusOK || f.store.Path() != other {
		t.Fatalf("status = %d, path = %s, body = %s", rec.Code, f.store.Path(), rec.Body.String())
	}
	if f.picker.shown {
		t.Error("the dialog was shown although a path was given")
	}
}

func TestOpenRejections(t *testing.T) {
	invalid := writeFile(t, "schema_version: 1\nnope: 1\n")
	tests := []struct {
		name     string
		body     string
		wantCode string
	}{
		{"invalid file", `{"path":"` + invalid + `"}`, "invalid_file"},
		{"missing file", `{"path":"/nowhere/x.yaml"}`, "invalid_file"},
		{"relative path", `{"path":"x.yaml"}`, "invalid_path"},
		{"empty path", `{"path":"  "}`, "invalid_path"},
		{"bad json", `{"path":`, "invalid_path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOpenFixture(t)

			rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/open", localOrigin, tt.body))

			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"`+tt.wantCode+`"`) {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if f.store.Path() != f.current {
				t.Error("switched despite the error")
			}
			if names := dirEntries(t, filepath.Dir(f.current)); len(names) != 1 {
				t.Errorf("files = %v, want no backup", names)
			}
		})
	}
}

func TestOpenRejectsForeignOrigin(t *testing.T) {
	f := newOpenFixture(t)
	f.picker.path = writeFile(t, "schema_version: 1\n")

	rec := serveReq(f.handlers, dataRequest(http.MethodPost, "/api/data/open", "http://evil.example", ""))

	if rec.Code != http.StatusForbidden || f.picker.shown {
		t.Errorf("status = %d, dialog shown = %v", rec.Code, f.picker.shown)
	}
}
