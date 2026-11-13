package loan

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
