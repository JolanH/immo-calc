package storage

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestReloadPicksUpAnExternalEdit(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nitems: []\n")
	if err := os.WriteFile(path, []byte("schema_version: 1\nitems:\n  - name: edited\n    qty: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := s.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if items, _ := Get[[]item](s, "items"); len(items) != 1 || items[0].Name != "edited" {
		t.Errorf("items = %+v", items)
	}
	// Saving after a reload must not be seen as a conflict.
	if err := Update(s, "items", addItem("b", 1)); err != nil {
		t.Errorf("save after reload: %v", err)
	}
}

func TestReloadOfAnInvalidFileKeepsTheCurrentData(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\nitems:\n  - name: kept\n    qty: 1\n")
	if err := os.WriteFile(path, []byte("schema_version: 1\nitems:\n  - name: bad\n    qty: -1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := s.Reload()

	var loadErr *LoadError
	if !errors.As(err, &loadErr) || loadErr.Line != 4 {
		t.Fatalf("err = %v, want a LoadError on line 4", err)
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 1 || items[0].Name != "kept" {
		t.Errorf("memory changed: %+v", items)
	}
}

func TestReloadEndpoint(t *testing.T) {
	s, path := loadedStore(t, "schema_version: 1\n")
	h := NewHandlers(s, HandlerConfig{CountKeys: []string{"items"}})
	origin := "http://127.0.0.1:8421"

	_ = os.WriteFile(path, []byte("schema_version: 1\nitems:\n  - name: a\n    qty: 1\n"), 0o644)
	ok := serveReq(h, dataRequest(http.MethodPost, "/api/data/reload", origin, ""))
	if ok.Code != http.StatusOK || !strings.Contains(ok.Body.String(), `"items":1`) {
		t.Errorf("valid reload: status = %d, body = %s", ok.Code, ok.Body.String())
	}

	_ = os.WriteFile(path, []byte("schema_version: 1\nnope: 1\n"), 0o644)
	bad := serveReq(h, dataRequest(http.MethodPost, "/api/data/reload", origin, ""))
	if bad.Code != http.StatusBadRequest || !strings.Contains(bad.Body.String(), `"invalid_file"`) {
		t.Errorf("invalid reload: status = %d, body = %s", bad.Code, bad.Body.String())
	}

	foreign := serveReq(h, dataRequest(http.MethodPost, "/api/data/reload", "http://evil.example", ""))
	if foreign.Code != http.StatusForbidden {
		t.Errorf("foreign origin: status = %d", foreign.Code)
	}
}
