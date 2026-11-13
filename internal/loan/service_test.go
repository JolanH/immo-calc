package loan

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"immo-calc/internal/storage"
)

var testNow = time.Date(2026, 11, 12, 20, 41, 0, 0, time.FixedZone("CET", 3600))

// newTestService returns a service on a fresh data file, with a fixed clock
// and predictable ids.
func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	return openService(t, path), path
}

func openService(t *testing.T, path string) *Service {
	t.Helper()
	store := storage.New()
	RegisterSection(store)
	if err := store.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	svc := NewService(store)
	svc.now = func() time.Time { return testNow }
	n := 0
	svc.newID = func() string {
		n++
		return strings.Repeat("0", 15) + string(rune('0'+n))
	}
	return svc
}

func TestCreateThenGetSurvivesARestart(t *testing.T) {
	svc, path := newTestService(t)

	created, err := svc.Create(validInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "0000000000000001" || !created.CreatedAt.Equal(testNow) || !created.UpdatedAt.Equal(testNow) {
		t.Errorf("created = %+v", created)
	}

	restarted := openService(t, path)
	got, err := restarted.Get(created.ID)
	if err != nil {
		t.Fatalf("get after restart: %v", err)
	}
	if got.Input != validInput() || !got.CreatedAt.Equal(testNow) {
		t.Errorf("after restart = %+v", got)
	}
	content, _ := os.ReadFile(path)
	for _, want := range []string{"loans:", "principal_cents: 20000000", "first_payment_month: 2026-12", "created_at: 2026-11-12T20:41:00+01:00"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("data file lacks %q:\n%s", want, content)
		}
	}
}

func TestCreateTrimsTheName(t *testing.T) {
	svc, _ := newTestService(t)
	in := validInput()
	in.Name = "  Prêt 1  "

	created, err := svc.Create(in)

	if err != nil || created.Name != "Prêt 1" {
		t.Errorf("created = %+v, err = %v", created, err)
	}
}

func TestCreateRejectsInvalidInputAndSavesNothing(t *testing.T) {
	svc, path := newTestService(t)
	in := validInput()
	in.PrincipalCents = 1

	_, err := svc.Create(in)

	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["principalCents"] == "" {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("the data file was written")
	}
}

func TestUpdate(t *testing.T) {
	svc, _ := newTestService(t)
	created, _ := svc.Create(validInput())
	later := testNow.Add(time.Hour)
	svc.now = func() time.Time { return later }
	in := validInput()
	in.AnnualRatePct = 3.1

	updated, err := svc.Update(created.ID, in)

	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.AnnualRatePct != 3.1 || !updated.CreatedAt.Equal(testNow) || !updated.UpdatedAt.Equal(later) {
		t.Errorf("updated = %+v", updated)
	}
	got, _ := svc.Get(created.ID)
	if got.AnnualRatePct != 3.1 {
		t.Errorf("stored rate = %v", got.AnnualRatePct)
	}
}

func TestUpdateErrors(t *testing.T) {
	svc, _ := newTestService(t)
	created, _ := svc.Create(validInput())

	if _, err := svc.Update("ffffffffffffffff", validInput()); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown id: err = %v", err)
	}
	bad := validInput()
	bad.DurationMonths = 18
	var verr *ValidationError
	if _, err := svc.Update(created.ID, bad); !errors.As(err, &verr) {
		t.Errorf("invalid input: err = %v", err)
	}
}

func TestGetUnknownLoan(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.Get("ffffffffffffffff"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestListSummaries(t *testing.T) {
	svc, _ := newTestService(t)
	empty, err := svc.List()
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty list = %+v, %v", empty, err)
	}
	first := validInput()
	first.Name = "Crédit Mutuel"
	second := validInput()
	second.Name = "BNP"
	second.PrincipalCents = 15_000_000
	second.AnnualRatePct = 4
	second.DurationMonths = 300
	second.InsuranceRatePct = 0
	second.ApplicationFeeCents = 0
	second.GuaranteeFeeCents = 0
	_, _ = svc.Create(first)
	_, _ = svc.Create(second)

	list, err := svc.List()

	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	want := Summary{
		ID: "0000000000000002", Name: "BNP", PrincipalCents: 15_000_000, AnnualRatePct: 4, DurationMonths: 300,
		MonthlyTotalCents: 79_176, TotalCostCents: 8_752_543,
	}
	if list[1] != want {
		t.Errorf("list[1] = %+v\nwant      %+v", list[1], want)
	}
	if list[0].MonthlyTotalCents != 120_992 || list[0].TotalCostCents != 9_388_066 {
		t.Errorf("list[0] = %+v", list[0])
	}
}

func TestDetailIncludesResults(t *testing.T) {
	svc, _ := newTestService(t)
	created, _ := svc.Create(validInput())

	detail, err := svc.Detail(created.ID)

	if err != nil || detail.ID != created.ID || detail.Result.TotalCostCents != 9_388_066 || len(detail.Result.Schedule) != 240 {
		t.Errorf("detail = %+v, err = %v", detail.Loan, err)
	}
}

func TestNewIDIs16HexChars(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		id := newRandomID()
		if !validID(id) || seen[id] {
			t.Fatalf("bad or repeated id %q", id)
		}
		seen[id] = true
	}
}
