package storage

import (
	"errors"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// itemsNeedSettings is a document check: every item's qty must not exceed
// settings.rate (a stand-in for "a property's loan_id must exist").
func itemsNeedSettings(get func(key string) []byte) error {
	var st settings
	_ = yaml.Unmarshal(get("settings"), &st)
	var items []item
	_ = yaml.Unmarshal(get("items"), &items)
	for i, it := range items {
		if float64(it.Qty) > st.Rate {
			return &DocumentError{Section: "items", Path: "[" + string(rune('0'+i)) + "].qty", Message: "quantité au-delà du taux pour « " + it.Name + " »"}
		}
	}
	return nil
}

func storeWithCheck() *Store {
	s := newTestStore()
	s.RegisterDocumentCheck(itemsNeedSettings)
	return s
}

const crossValid = "schema_version: 1\nsettings:\n  rate: 5\nitems:\n  - name: a\n    qty: 2\n"
const crossInvalid = "schema_version: 1\nsettings:\n  rate: 5\nitems:\n  - name: a\n    qty: 2\n  - name: b\n    qty: 9\n"

func TestDocumentCheckPassesOnAValidFile(t *testing.T) {
	if err := storeWithCheck().Load(writeFile(t, crossValid)); err != nil {
		t.Fatalf("load: %v", err)
	}
}

func TestDocumentCheckFailureNamesTheLine(t *testing.T) {
	path := writeFile(t, crossInvalid)

	err := storeWithCheck().Load(path)

	var loadErr *LoadError
	if !errors.As(err, &loadErr) || loadErr.Line != 8 || !strings.Contains(err.Error(), "« b »") {
		t.Errorf("err = %v", err)
	}
}

func TestDocumentCheckGuardsImportOpenAndReload(t *testing.T) {
	s := storeWithCheck()
	if err := s.Load(writeFile(t, crossValid)); err != nil {
		t.Fatal(err)
	}

	if err := s.Import([]byte(crossInvalid)); err == nil {
		t.Error("import accepted an inconsistent document")
	}
	if err := s.SwitchTo(writeFile(t, crossInvalid)); err == nil {
		t.Error("switch accepted an inconsistent document")
	}
}

func TestDocumentCheckRunsBeforeASave(t *testing.T) {
	s := storeWithCheck()
	path := writeFile(t, crossValid)
	_ = s.Load(path)

	err := Update(s, "items", addItem("c", 50))

	var docErr *DocumentError
	if !errors.As(err, &docErr) {
		t.Fatalf("err = %v, want a DocumentError", err)
	}
	if items, _ := Get[[]item](s, "items"); len(items) != 1 {
		t.Errorf("memory changed: %+v", items)
	}
}

func TestPlainDocumentCheckErrorsAreReported(t *testing.T) {
	s := newTestStore()
	s.RegisterDocumentCheck(func(func(string) []byte) error { return errors.New("boom") })

	if err := s.Load(writeFile(t, crossValid)); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v", err)
	}
}
