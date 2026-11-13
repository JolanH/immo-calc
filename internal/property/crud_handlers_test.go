package property

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func apiRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "127.0.0.1:8421"
	req.Header.Set("Origin", localOrigin)
	return req
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return v
}

const createPropertyBody = `{"name":"T2 Lyon 7e","priceCents":15000000,"dpe":"D","monthlyRentCents":70000,
	"monthlyRecoverableChargesCents":5000,"annualPropertyTaxCents":90000,"annualCondoFeesCents":120000,
	"annualLandlordInsuranceCents":15000,"agencyManaged":false,"taxRegime":"micro_foncier","loanId":""}`

type propertyJSON struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Result struct {
		NetAnnualIncomeCents int64 `json:"netAnnualIncomeCents"`
	} `json:"result"`
	Loan *LoanRef `json:"loan"`
}

func TestPropertyCRUDEndpoints(t *testing.T) {
	f := newFixture(t)
	h := NewHandlers(f.svc)

	created := serveProperty(h, apiRequest(http.MethodPost, "/api/properties", createPropertyBody))
	if created.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", created.Code, created.Body.String())
	}
	c := decodeBody[propertyJSON](t, created)
	if c.ID == "" || c.Result.NetAnnualIncomeCents != 285_592 || c.Loan != nil {
		t.Errorf("created = %+v", c)
	}

	got := serveProperty(h, apiRequest(http.MethodGet, "/api/properties/"+c.ID, ""))
	if got.Code != http.StatusOK || decodeBody[propertyJSON](t, got).Name != "T2 Lyon 7e" {
		t.Errorf("get: status = %d, body = %.200s", got.Code, got.Body.String())
	}

	updated := serveProperty(h, apiRequest(http.MethodPut, "/api/properties/"+c.ID, strings.Replace(createPropertyBody, "T2 Lyon 7e", "T2 Lyon", 1)))
	if updated.Code != http.StatusOK || decodeBody[propertyJSON](t, updated).Name != "T2 Lyon" {
		t.Errorf("update: status = %d, body = %.200s", updated.Code, updated.Body.String())
	}

	dup := serveProperty(h, apiRequest(http.MethodPost, "/api/properties/"+c.ID+"/duplicate", ""))
	if dup.Code != http.StatusCreated || decodeBody[propertyJSON](t, dup).Name != "T2 Lyon (copie)" {
		t.Errorf("duplicate: status = %d, body = %.200s", dup.Code, dup.Body.String())
	}

	list := serveProperty(h, apiRequest(http.MethodGet, "/api/properties", ""))
	items := decodeBody[[]map[string]any](t, list)
	if list.Code != http.StatusOK || len(items) != 2 || items[0]["netAfterTaxYieldPct"] != 1.75 {
		t.Errorf("list: status = %d, body = %s", list.Code, list.Body.String())
	}

	del := serveProperty(h, apiRequest(http.MethodDelete, "/api/properties/"+c.ID, ""))
	if del.Code != http.StatusNoContent {
		t.Errorf("delete: status = %d", del.Code)
	}
}

func TestPropertyListFilterAndEmptyList(t *testing.T) {
	f := newFixture(t)
	h := NewHandlers(f.svc)
	if rec := serveProperty(h, apiRequest(http.MethodGet, "/api/properties", "")); rec.Body.String() != "[]\n" {
		t.Errorf("empty list = %q", rec.Body.String())
	}
	l := saveLoan(t, f, "Petit prêt")
	serveProperty(h, apiRequest(http.MethodPost, "/api/properties", createPropertyBody))
	serveProperty(h, apiRequest(http.MethodPost, "/api/properties", strings.Replace(createPropertyBody, `"loanId":""`, `"loanId":"`+l.ID+`"`, 1)))

	rec := serveProperty(h, apiRequest(http.MethodGet, "/api/properties?loanId="+l.ID, ""))

	items := decodeBody[[]map[string]any](t, rec)
	if len(items) != 1 || items[0]["loanName"] != "Petit prêt" {
		t.Errorf("filtered = %s", rec.Body.String())
	}
}

func TestEvaluateExcludesTheEditedProperty(t *testing.T) {
	f := newFixture(t)
	h := NewHandlers(f.svc)
	c := decodeBody[propertyJSON](t, serveProperty(h, apiRequest(http.MethodPost, "/api/properties", createPropertyBody)))

	asNew := serveProperty(h, evaluateRequest(createPropertyBody))
	asEdit := serveProperty(h, apiRequest(http.MethodPost, "/api/properties/evaluate?id="+c.ID, createPropertyBody))

	if !strings.Contains(asNew.Body.String(), "micro_ceiling") {
		t.Errorf("new: %s", asNew.Body.String())
	}
	if strings.Contains(asEdit.Body.String(), "micro_ceiling") {
		t.Errorf("edit: %s", asEdit.Body.String())
	}
}

func TestPropertyEndpointErrors(t *testing.T) {
	f := newFixture(t)
	h := NewHandlers(f.svc)
	tests := []struct {
		name     string
		req      *http.Request
		wantCode int
		wantBody string
	}{
		{"create invalid", apiRequest(http.MethodPost, "/api/properties", strings.Replace(createPropertyBody, `"dpe":"D"`, `"dpe":""`, 1)), 400, `"dpe"`},
		{"create without name", apiRequest(http.MethodPost, "/api/properties", strings.Replace(createPropertyBody, `"T2 Lyon 7e"`, `""`, 1)), 400, `"name"`},
		{"create malformed", apiRequest(http.MethodPost, "/api/properties", `{`), 400, "invalid_json"},
		{"get unknown", apiRequest(http.MethodGet, "/api/properties/ffffffffffffffff", ""), 404, "Bien introuvable"},
		{"update unknown", apiRequest(http.MethodPut, "/api/properties/ffffffffffffffff", createPropertyBody), 404, "Bien introuvable"},
		{"delete unknown", apiRequest(http.MethodDelete, "/api/properties/ffffffffffffffff", ""), 404, "Bien introuvable"},
		{"duplicate unknown", apiRequest(http.MethodPost, "/api/properties/ffffffffffffffff/duplicate", ""), 404, "Bien introuvable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveProperty(h, tt.req)
			if rec.Code != tt.wantCode || !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPropertyWritesRequireSameOrigin(t *testing.T) {
	h := NewHandlers(newFixture(t).svc)
	for _, req := range []*http.Request{
		apiRequest(http.MethodPost, "/api/properties", createPropertyBody),
		apiRequest(http.MethodPut, "/api/properties/0000000000000001", createPropertyBody),
		apiRequest(http.MethodDelete, "/api/properties/0000000000000001", ""),
		apiRequest(http.MethodPost, "/api/properties/0000000000000001/duplicate", ""),
	} {
		req.Header.Set("Origin", "http://evil.example")
		if rec := serveProperty(h, req); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: status = %d", req.Method, req.URL.Path, rec.Code)
		}
	}
}

func TestPropertySaveConflictAnswers409(t *testing.T) {
	f := newFixture(t)
	h := NewHandlers(f.svc)
	serveProperty(h, apiRequest(http.MethodPost, "/api/properties", createPropertyBody))
	if err := os.WriteFile(f.path, []byte("schema_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := serveProperty(h, apiRequest(http.MethodPost, "/api/properties", createPropertyBody))

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestListWithActiveLoanEndpoint(t *testing.T) {
	f := newFixture(t)
	h := NewHandlers(f.svc)
	active := saveLoan(t, f, "Prêt type")
	serveProperty(h, apiRequest(http.MethodPost, "/api/properties", createPropertyBody))

	rec := serveProperty(h, apiRequest(http.MethodGet, "/api/properties?activeLoanId="+active.ID, ""))
	items := decodeBody[[]map[string]any](t, rec)
	if rec.Code != http.StatusOK || len(items) != 1 || items[0]["loanName"] != "Prêt type" || items[0]["cashFlowAfterTaxCents"] == nil {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	unknown := serveProperty(h, apiRequest(http.MethodGet, "/api/properties?activeLoanId=ffffffffffffffff", ""))
	if unknown.Code != http.StatusNotFound || !strings.Contains(unknown.Body.String(), "Prêt introuvable") {
		t.Errorf("unknown: status = %d, body = %s", unknown.Code, unknown.Body.String())
	}
}
