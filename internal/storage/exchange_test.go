package storage

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 11, 13, 14, 5, 9, 0, time.Local)

func clockAt(t time.Time) func() time.Time { return func() time.Time { return t } }

const validImport = "schema_version: 1\nitems:\n  - name: imported\n    qty: 7\n"

func TestExportThenImportRoundTrip(t *testing.T) {
	src, _ := loadedStore(t, "schema_version: 1\nsettings:\n  rate: 4.2\nitems:\n  - name: a\n    qty: 1\n")
	exported, err := src.Export()
	if err != nil {
		t.Fatal(err)
	}
	dst, _ := loadedStore(t, "schema_version: 1\n")

	if err := dst.Import(exported); err != nil {
		t.Fatalf("import: %v", err)
	}

	items, _ := Get[[]item](dst, "items")
	st, _ := Get[settings](dst, "settings")
	if len(items) != 1 || items[0] != (item{"a", 1}) || st.Rate != 4.2 {
		t.Errorf("imported items = %+v, settings = %+v", items, st)
	}
	again, _ := dst.Export()
	if string(again) != string(exported) {
		t.Errorf("export after import differs:\n%s\n---\n%s", exported, again)
	}
}

func TestImportReplacesDataAndBacksUpTheCurrentFile(t *testing.T) {
	original := "schema_version: 1\nitems:\n  - name: old\n    qty: 1\n"
	s, path := loadedStore(t, original)
	s.now = clockAt(fixedNow)

	if err := s.Import([]byte(validImport)); err != nil {
		t.Fatalf("import: %v", err)
	}

	backup := filepath.Join(filepath.Dir(path), "immo-calc.bak-20261113-140509.yaml")
	if got, err := os.ReadFile(backup); err != nil || string(got) != original {
		t.Errorf("backup = %q, err = %v", got, err)
	}
	if names := dirEntries(t, filepath.Dir(path)); len(names) != 2 {
		t.Errorf("files = %v, want the data file and exactly one backup", names)
	}
	reloaded := newTestStore()
	_ = reloaded.Load(path)
	if items, _ := Get[[]item](reloaded, "items"); len(items) != 1 || items[0].Name != "imported" {
		t.Errorf("data file not replaced: %+v", items)
	}
}

func TestImportWithoutAnExistingFileMakesNoBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	s := newTestStore()
	_ = s.Load(path)

	if err := s.Import([]byte(validImport)); err != nil {
		t.Fatal(err)
	}

	if names := dirEntries(t, filepath.Dir(path)); len(names) != 1 {
		t.Errorf("files = %v, want only the new data file", names)
	}
}

func TestInvalidImportChangesNothing(t *testing.T) {
	for name, content := range map[string]string{
		"malformed":       "schema_version: 1\nitems: [\n",
		"invalid section": "schema_version: 1\nitems:\n  - name: x\n    qty: -1\n",
		"unknown key":     "schema_version: 1\nfoo: 1\n",
		"too new":         "schema_version: 9\n",
	} {
		t.Run(name, func(t *testing.T) {
			original := "schema_version: 1\nitems:\n  - name: old\n    qty: 1\n"
			s, path := loadedStore(t, original)

			err := s.Import([]byte(content))

			var loadErr *LoadError
			if !errors.As(err, &loadErr) || !strings.HasPrefix(err.Error(), "Le fichier importé est invalide") {
				t.Fatalf("err = %v", err)
			}
			if now, _ := os.ReadFile(path); string(now) != original {
				t.Error("data file changed")
			}
			if names := dirEntries(t, filepath.Dir(path)); len(names) != 1 {
				t.Errorf("files = %v, want no backup", names)
			}
			if items, _ := Get[[]item](s, "items"); len(items) != 1 || items[0].Name != "old" {
				t.Errorf("memory changed: %+v", items)
			}
		})
	}
}

func TestBackupNameDoesNotOverwriteAnEarlierBackup(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\n")
	s.now = clockAt(fixedNow)

	if err := s.Import([]byte(validImport)); err != nil {
		t.Fatal(err)
	}
	if err := s.Import([]byte("schema_version: 1\n")); err != nil {
		t.Fatal(err)
	}

	if names := dirEntries(t, filepath.Dir(path)); len(names) != 3 {
		t.Errorf("files = %v, want the data file and two backups", names)
	}
}

func dataRequest(method, path, origin, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "127.0.0.1:8421"
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}

func serveReq(h *Handlers, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestExportEndpoint(t *testing.T) {
	s, _ := loadedStore(t, "schema_version: 1\nitems:\n  - name: a\n    qty: 1\n")
	s.now = clockAt(fixedNow)
	h := NewHandlers(s, HandlerConfig{})

	rec := serveReq(h, dataRequest(http.MethodGet, "/api/data/export", "", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="immo-calc-export-2026-11-13.yaml"` {
		t.Errorf("content disposition = %q", cd)
	}
	if !strings.Contains(rec.Body.String(), "name: a") || !strings.Contains(rec.Body.String(), "schema_version: 1") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestImportEndpoint(t *testing.T) {
	s, _ := loadedStore(t, "schema_version: 1\n")
	h := NewHandlers(s, HandlerConfig{CountKeys: []string{"items"}})

	rec := serveReq(h, dataRequest(http.MethodPost, "/api/data/import", "http://127.0.0.1:8421", validImport))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body infoBody
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Counts["items"] != 1 {
		t.Errorf("counts = %v", body.Counts)
	}
}

func TestImportEndpointRejectsInvalidFiles(t *testing.T) {
	s, _ := loadedStore(t, "schema_version: 1\n")
	h := NewHandlers(s, HandlerConfig{})

	rec := serveReq(h, dataRequest(http.MethodPost, "/api/data/import", "http://127.0.0.1:8421", "nope: 1\n"))

	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_import"`) ||
		!strings.Contains(rec.Body.String(), "Le fichier importé est invalide") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestImportEndpointRejectsForeignOrigin(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\n")
	h := NewHandlers(s, HandlerConfig{})

	rec := serveReq(h, dataRequest(http.MethodPost, "/api/data/import", "http://evil.example", validImport))

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d", rec.Code)
	}
	if names := dirEntries(t, filepath.Dir(path)); len(names) != 1 {
		t.Errorf("files = %v", names)
	}
}
