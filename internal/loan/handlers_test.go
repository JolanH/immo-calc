package loan

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const localOrigin = "http://127.0.0.1:8421"

func apiRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "127.0.0.1:8421"
	req.Header.Set("Origin", localOrigin)
	return req
}

func serveLoan(h *Handlers, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const simulateBody = `{"principalCents":20000000,"annualRatePct":3.5,"durationMonths":240,
	"insuranceRatePct":0.3,"applicationFeeCents":100000,"guaranteeFeeCents":250000,"firstPaymentMonth":"2026-12"}`

func TestSimulateReturnsTheDocumentedJSON(t *testing.T) {
	rec := serveLoan(NewHandlers(nil), apiRequest(http.MethodPost, "/api/loans/simulate", simulateBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	// Totals from the result JSON example in SPEC-loan.md.
	for key, want := range map[string]float64{
		"monthlyPaymentCents": 115992, "monthlyInsuranceCents": 5000, "monthlyTotalCents": 120992,
		"totalInterestCents": 7838066, "totalInsuranceCents": 1200000, "totalFeesCents": 350000,
		"totalCostCents": 9388066, "totalRepaidCents": 29038066,
	} {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
	schedule := body["schedule"].([]any)
	first := schedule[0].(map[string]any)
	wantFirst := map[string]any{"index": 1.0, "month": "2026-12", "paymentCents": 115992.0, "interestCents": 58333.0,
		"principalCents": 57659.0, "insuranceCents": 5000.0, "remainingCents": 19942341.0}
	for k, v := range wantFirst {
		if first[k] != v {
			t.Errorf("schedule[0].%s = %v, want %v", k, first[k], v)
		}
	}
	if len(schedule) != 240 {
		t.Errorf("schedule length = %d", len(schedule))
	}
	yearly := body["yearly"].([]any)
	firstYear := yearly[0].(map[string]any)
	if firstYear["year"] != 2026.0 || firstYear["installments"] != 1.0 {
		t.Errorf("yearly[0] = %v", firstYear)
	}
}

func TestSimulateDoesNotNeedAName(t *testing.T) {
	rec := serveLoan(NewHandlers(nil), apiRequest(http.MethodPost, "/api/loans/simulate", simulateBody))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestSimulateReturnsFieldErrors(t *testing.T) {
	body := `{"principalCents":5000,"annualRatePct":3.5,"durationMonths":18,"firstPaymentMonth":"2026-12"}`

	rec := serveLoan(NewHandlers(nil), apiRequest(http.MethodPost, "/api/loans/simulate", body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Error.Code != "validation_failed" || resp.Error.Fields["principalCents"] == "" ||
		resp.Error.Fields["durationMonths"] == "" || len(resp.Error.Fields) != 2 {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestSimulateRejectsMalformedJSON(t *testing.T) {
	for _, body := range []string{`{"principalCents":`, `{"principalCents":"abc"}`, ``} {
		rec := serveLoan(NewHandlers(nil), apiRequest(http.MethodPost, "/api/loans/simulate", body))
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_json"`) {
			t.Errorf("body %q: status = %d, %s", body, rec.Code, rec.Body.String())
		}
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return v
}

const createBody = `{"name":"Appart Lyon","principalCents":20000000,"annualRatePct":3.5,"durationMonths":240,
	"insuranceRatePct":0.3,"applicationFeeCents":100000,"guaranteeFeeCents":250000,"firstPaymentMonth":"2026-12"}`

type detailJSON struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	PrincipalCents    int64  `json:"principalCents"`
	FirstPaymentMonth string `json:"firstPaymentMonth"`
	CreatedAt         string `json:"createdAt"`
	Result            struct {
		TotalCostCents int64            `json:"totalCostCents"`
		Schedule       []map[string]any `json:"schedule"`
	} `json:"result"`
}

func TestLoanCRUDEndpoints(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)

	created := serveLoan(h, apiRequest(http.MethodPost, "/api/loans", createBody))
	if created.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", created.Code, created.Body.String())
	}
	c := decode[detailJSON](t, created)
	if c.ID == "" || c.Name != "Appart Lyon" || c.FirstPaymentMonth != "2026-12" || c.CreatedAt == "" ||
		c.Result.TotalCostCents != 9_388_066 || len(c.Result.Schedule) != 240 {
		t.Errorf("created = %+v", c)
	}

	got := serveLoan(h, apiRequest(http.MethodGet, "/api/loans/"+c.ID, ""))
	if got.Code != http.StatusOK || decode[detailJSON](t, got).Result.TotalCostCents != 9_388_066 {
		t.Errorf("get: status = %d, body = %.200s", got.Code, got.Body.String())
	}

	updatedBody := strings.Replace(createBody, `"Appart Lyon"`, `"Appart Lyon v2"`, 1)
	updated := serveLoan(h, apiRequest(http.MethodPut, "/api/loans/"+c.ID, updatedBody))
	if updated.Code != http.StatusOK || decode[detailJSON](t, updated).Name != "Appart Lyon v2" {
		t.Errorf("update: status = %d, body = %.200s", updated.Code, updated.Body.String())
	}

	list := serveLoan(h, apiRequest(http.MethodGet, "/api/loans", ""))
	items := decode[[]map[string]any](t, list)
	if list.Code != http.StatusOK || len(items) != 1 || items[0]["name"] != "Appart Lyon v2" ||
		items[0]["monthlyTotalCents"] != 120992.0 || items[0]["totalCostCents"] != 9388066.0 {
		t.Errorf("list: status = %d, body = %s", list.Code, list.Body.String())
	}
}

func TestEmptyLoanListIsAnArray(t *testing.T) {
	svc, _ := newTestService(t)
	rec := serveLoan(NewHandlers(svc), apiRequest(http.MethodGet, "/api/loans", ""))
	if rec.Body.String() != "[]\n" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestLoanEndpointErrors(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)
	invalid := strings.Replace(createBody, `"durationMonths":240`, `"durationMonths":18`, 1)

	tests := []struct {
		name     string
		req      *http.Request
		wantCode int
		wantBody string
	}{
		{"create invalid", apiRequest(http.MethodPost, "/api/loans", invalid), 400, `"durationMonths"`},
		{"create without name", apiRequest(http.MethodPost, "/api/loans", strings.Replace(createBody, `"Appart Lyon"`, `""`, 1)), 400, `"name"`},
		{"create malformed", apiRequest(http.MethodPost, "/api/loans", `{`), 400, `"invalid_json"`},
		{"get unknown", apiRequest(http.MethodGet, "/api/loans/ffffffffffffffff", ""), 404, "Prêt introuvable"},
		{"update unknown", apiRequest(http.MethodPut, "/api/loans/ffffffffffffffff", createBody), 404, "Prêt introuvable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveLoan(h, tt.req)
			if rec.Code != tt.wantCode || !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestLoanWritesRequireSameOrigin(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)
	for _, req := range []*http.Request{
		apiRequest(http.MethodPost, "/api/loans", createBody),
		apiRequest(http.MethodPut, "/api/loans/0000000000000001", createBody),
	} {
		req.Header.Set("Origin", "http://evil.example")
		if rec := serveLoan(h, req); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: status = %d", req.Method, req.URL.Path, rec.Code)
		}
	}
}

func TestLoanSaveConflictAnswers409(t *testing.T) {
	svc, path := newTestService(t)
	h := NewHandlers(svc)
	serveLoan(h, apiRequest(http.MethodPost, "/api/loans", createBody))
	if err := os.WriteFile(path, []byte("schema_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := serveLoan(h, apiRequest(http.MethodPost, "/api/loans", createBody))

	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "file_changed_externally") {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDuplicateAndDeleteEndpoints(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)
	original := decode[detailJSON](t, serveLoan(h, apiRequest(http.MethodPost, "/api/loans", createBody)))

	dup := serveLoan(h, apiRequest(http.MethodPost, "/api/loans/"+original.ID+"/duplicate", ""))
	if dup.Code != http.StatusCreated || decode[detailJSON](t, dup).Name != "Appart Lyon (copie)" {
		t.Fatalf("duplicate: status = %d, body = %.200s", dup.Code, dup.Body.String())
	}

	del := serveLoan(h, apiRequest(http.MethodDelete, "/api/loans/"+original.ID, ""))
	if del.Code != http.StatusNoContent || del.Body.Len() != 0 {
		t.Errorf("delete: status = %d, body = %q", del.Code, del.Body.String())
	}
	if again := serveLoan(h, apiRequest(http.MethodDelete, "/api/loans/"+original.ID, "")); again.Code != http.StatusNotFound {
		t.Errorf("delete again: status = %d", again.Code)
	}
	if unknown := serveLoan(h, apiRequest(http.MethodPost, "/api/loans/ffffffffffffffff/duplicate", "")); unknown.Code != http.StatusNotFound {
		t.Errorf("duplicate unknown: status = %d", unknown.Code)
	}
}

func TestDeleteRefusedByAGuardAnswers409(t *testing.T) {
	svc, _ := newTestService(t)
	svc.RegisterDeleteGuard(func(string) error { return errors.New("Ce prêt est utilisé par le bien « T2 Lyon 7e »") })
	h := NewHandlers(svc)
	l := decode[detailJSON](t, serveLoan(h, apiRequest(http.MethodPost, "/api/loans", createBody)))

	rec := serveLoan(h, apiRequest(http.MethodDelete, "/api/loans/"+l.ID, ""))

	resp := decode[struct {
		Error struct{ Code, Message string } `json:"error"`
	}](t, rec)
	if rec.Code != http.StatusConflict || resp.Error.Code != "loan_in_use" ||
		resp.Error.Message != "Ce prêt est utilisé par le bien « T2 Lyon 7e »" {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDuplicateAndDeleteRequireSameOrigin(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)
	for _, req := range []*http.Request{
		apiRequest(http.MethodPost, "/api/loans/0000000000000001/duplicate", ""),
		apiRequest(http.MethodDelete, "/api/loans/0000000000000001", ""),
	} {
		req.Header.Set("Origin", "http://evil.example")
		if rec := serveLoan(h, req); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: status = %d", req.Method, req.URL.Path, rec.Code)
		}
	}
}
