package loan

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestActiveLoanStartsEmpty(t *testing.T) {
	svc, _ := newTestService(t)

	if id := svc.ActiveLoan(); id != "" {
		t.Errorf("active = %q, want none", id)
	}
}

func TestSetActiveLoan(t *testing.T) {
	svc, _ := newTestService(t)
	l, _ := svc.Create(validInput())

	if err := svc.SetActiveLoan(l.ID); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := svc.ActiveLoan(); got != l.ID {
		t.Errorf("active = %q", got)
	}
	if err := svc.SetActiveLoan(""); err != nil || svc.ActiveLoan() != "" {
		t.Errorf("clear: err = %v, active = %q", err, svc.ActiveLoan())
	}
}

func TestSetActiveLoanRejectsUnknownLoans(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.SetActiveLoan("ffffffffffffffff"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestActiveLoanIsNotSaved(t *testing.T) {
	svc, path := newTestService(t)
	l, _ := svc.Create(validInput())
	_ = svc.SetActiveLoan(l.ID)

	restarted := openService(t, path)

	if got := restarted.ActiveLoan(); got != "" {
		t.Errorf("active after restart = %q, want none", got)
	}
}

func TestDeletingTheActiveLoanClearsIt(t *testing.T) {
	svc, _ := newTestService(t)
	active, _ := svc.Create(validInput())
	other, _ := svc.Create(validInput())
	_ = svc.SetActiveLoan(active.ID)

	_ = svc.Delete(other.ID)
	if svc.ActiveLoan() != active.ID {
		t.Fatal("deleting another loan cleared the active one")
	}
	_ = svc.Delete(active.ID)
	if got := svc.ActiveLoan(); got != "" {
		t.Errorf("active = %q after deleting it", got)
	}
}

func TestActiveLoanEndpoints(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)
	l, _ := svc.Create(validInput())

	if rec := serveLoan(h, apiRequest(http.MethodGet, "/api/active-loan", "")); rec.Body.String() != `{"loanId":null}`+"\n" {
		t.Errorf("initial: %s", rec.Body.String())
	}
	put := serveLoan(h, apiRequest(http.MethodPut, "/api/active-loan", `{"loanId":"`+l.ID+`"}`))
	if put.Code != http.StatusOK || put.Body.String() != `{"loanId":"`+l.ID+`"}`+"\n" {
		t.Errorf("set: status = %d, body = %s", put.Code, put.Body.String())
	}
	if rec := serveLoan(h, apiRequest(http.MethodGet, "/api/active-loan", "")); !strings.Contains(rec.Body.String(), l.ID) {
		t.Errorf("after set: %s", rec.Body.String())
	}
	clear := serveLoan(h, apiRequest(http.MethodPut, "/api/active-loan", `{"loanId":null}`))
	if clear.Body.String() != `{"loanId":null}`+"\n" {
		t.Errorf("clear: %s", clear.Body.String())
	}
}

func TestActiveLoanEndpointErrors(t *testing.T) {
	svc, _ := newTestService(t)
	h := NewHandlers(svc)

	unknown := serveLoan(h, apiRequest(http.MethodPut, "/api/active-loan", `{"loanId":"ffffffffffffffff"}`))
	if unknown.Code != http.StatusNotFound {
		t.Errorf("unknown: status = %d", unknown.Code)
	}
	malformed := serveLoan(h, apiRequest(http.MethodPut, "/api/active-loan", `{`))
	if malformed.Code != http.StatusBadRequest {
		t.Errorf("malformed: status = %d", malformed.Code)
	}
	foreign := apiRequest(http.MethodPut, "/api/active-loan", `{"loanId":null}`)
	foreign.Header.Set("Origin", "http://evil.example")
	if rec := serveLoan(h, foreign); rec.Code != http.StatusForbidden {
		t.Errorf("foreign origin: status = %d", rec.Code)
	}
}
