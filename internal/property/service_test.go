package property

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"immo-calc/internal/loan"
	"immo-calc/internal/storage"
)

var propNow = time.Date(2026, 11, 13, 10, 0, 0, 0, time.FixedZone("CET", 3600))

type fixture struct {
	path  string
	store *storage.Store
	loans *loan.Service
	svc   *Service
}

// newFixture opens real loan and property services on a fresh data file.
func newFixture(t *testing.T) fixture {
	t.Helper()
	return reopen(t, filepath.Join(t.TempDir(), "immo-calc.yaml"))
}

func reopen(t *testing.T, path string) fixture {
	t.Helper()
	store := openStore(t, path)
	loans := loan.NewService(store)
	svc := NewService(store, loans)
	svc.now = func() time.Time { return propNow }
	return fixture{path, store, loans, svc}
}

func saveLoan(t *testing.T, f fixture, name string) loan.Loan {
	t.Helper()
	l, err := f.loans.Create(loan.Input{
		Name: name, PrincipalCents: 5_000_000, AnnualRatePct: 3, DurationMonths: 240,
		InsuranceRatePct: 0.3, ApplicationFeeCents: 100_000, FirstPaymentMonth: "2026-12",
	})
	if err != nil {
		t.Fatalf("create loan: %v", err)
	}
	return l
}

func TestCreatePropertySurvivesARestart(t *testing.T) {
	f := newFixture(t)

	created, err := f.svc.Create(validPropertyInput())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(created.ID) != 16 || !created.CreatedAt.Equal(propNow) {
		t.Errorf("created = %+v", created)
	}

	detail, err := reopen(t, f.path).svc.Detail(created.ID)
	if err != nil {
		t.Fatalf("detail after restart: %v", err)
	}
	if detail.Input != validPropertyInput() || detail.Result.NetAnnualIncomeCents != 285_592 {
		t.Errorf("after restart = %+v / %+v", detail.Input, detail.Result)
	}
	content, _ := os.ReadFile(f.path)
	for _, want := range []string{"properties:", "price_cents: 15000000", "dpe: D", "tax_regime: micro_foncier", "agency_managed: false"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("data file lacks %q:\n%s", want, content)
		}
	}
	if strings.Contains(string(content), "loan_id") {
		t.Errorf("empty loan_id written:\n%s", content)
	}
}

func TestCreatePropertyRejectsInvalidInput(t *testing.T) {
	f := newFixture(t)
	in := validPropertyInput()
	in.LoanID = "ffffffffffffffff"

	_, err := f.svc.Create(in)

	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["loanId"] == "" {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateDuplicateDeleteProperty(t *testing.T) {
	f := newFixture(t)
	p, _ := f.svc.Create(validPropertyInput())
	later := propNow.Add(time.Hour)
	f.svc.now = func() time.Time { return later }

	in := validPropertyInput()
	in.MonthlyRentCents = 75_000
	updated, err := f.svc.Update(p.ID, in)
	if err != nil || updated.MonthlyRentCents != 75_000 || !updated.UpdatedAt.Equal(later) || !updated.CreatedAt.Equal(propNow) {
		t.Fatalf("update = %+v, %v", updated, err)
	}

	copied, err := f.svc.Duplicate(p.ID)
	if err != nil || copied.Name != "T2 Lyon 7e (copie)" || copied.ID == p.ID || copied.MonthlyRentCents != 75_000 {
		t.Fatalf("duplicate = %+v, %v", copied, err)
	}

	if err := f.svc.Delete(p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := f.svc.Get(p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted property still there: %v", err)
	}
	for _, op := range []func() error{
		func() error { _, err := f.svc.Update(p.ID, in); return err },
		func() error { _, err := f.svc.Duplicate(p.ID); return err },
		func() error { return f.svc.Delete(p.ID) },
		func() error { _, err := f.svc.Detail(p.ID); return err },
	} {
		if err := op(); !errors.Is(err, ErrNotFound) {
			t.Errorf("unknown id: err = %v", err)
		}
	}
}

func TestDeletingAPropertyKeepsItsLoan(t *testing.T) {
	f := newFixture(t)
	l := saveLoan(t, f, "Petit prêt")
	in := validPropertyInput()
	in.LoanID = l.ID
	p, _ := f.svc.Create(in)

	if err := f.svc.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.loans.Get(l.ID); err != nil {
		t.Errorf("loan gone: %v", err)
	}
}

func TestListSummaries(t *testing.T) {
	f := newFixture(t)
	l := saveLoan(t, f, "Petit prêt")
	plain := validPropertyInput()
	financed := validPropertyInput()
	financed.Name = "Studio"
	financed.TaxRegime = ReelFoncier
	financed.LoanID = l.ID
	_, _ = f.svc.Create(plain)
	_, _ = f.svc.Create(financed)

	list, err := f.svc.List("")

	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	first := list[0]
	if first.Name != "T2 Lyon 7e" || first.PriceCents != 15_000_000 || first.MonthlyRentCents != 70_000 || first.DPE != "D" ||
		first.NetAfterTaxYieldPct != 1.75 || first.MonthlyNetIncomeCents != 23_799 || first.CashFlowAfterTaxCents != nil ||
		first.LoanID != "" || first.LoanName != "" || first.TaxRegime != MicroFoncier {
		t.Errorf("list[0] = %+v", first)
	}
	second := list[1]
	if second.TaxRegime != ReelFoncier || second.LoanID != l.ID || second.LoanName != "Petit prêt" || second.CashFlowAfterTaxCents == nil || *second.CashFlowAfterTaxCents != 1_170 {
		t.Errorf("list[1] = %+v", second)
	}
}

func TestListFilteredByLoan(t *testing.T) {
	f := newFixture(t)
	l := saveLoan(t, f, "Petit prêt")
	financed := validPropertyInput()
	financed.LoanID = l.ID
	_, _ = f.svc.Create(validPropertyInput())
	_, _ = f.svc.Create(financed)

	list, _ := f.svc.List(l.ID)

	if len(list) != 1 || list[0].LoanID != l.ID {
		t.Errorf("filtered list = %+v", list)
	}
}

func TestDetailIncludesTheLinkedLoan(t *testing.T) {
	f := newFixture(t)
	l := saveLoan(t, f, "Petit prêt")
	in := validPropertyInput()
	in.TaxRegime = ReelFoncier
	in.LoanID = l.ID
	p, _ := f.svc.Create(in)

	d, _ := f.svc.Detail(p.ID)

	if d.Loan == nil || d.Loan.ID != l.ID || d.Loan.Name != "Petit prêt" {
		t.Errorf("loan = %+v", d.Loan)
	}
	if d.Result.Financing == nil || d.Result.TaxableIncomeCents != 377_534 {
		t.Errorf("result = %+v", d.Result)
	}
}

func TestChangingASettingChangesEveryPropertysResults(t *testing.T) {
	f := newFixture(t)
	a, _ := f.svc.Create(validPropertyInput())
	b, _ := f.svc.Create(validPropertyInput())
	s := DefaultSettings()
	s.NotaryFeePct = 0
	if _, err := f.svc.UpdateSettings(s); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{a.ID, b.ID} {
		d, _ := f.svc.Detail(id)
		if d.Result.AcquisitionCostCents != 15_000_000 {
			t.Errorf("%s: A = %d after setting notary to 0", id, d.Result.AcquisitionCostCents)
		}
	}
}

func TestMicroCeilingWarningSumsAllPropertiesOfTheRegime(t *testing.T) {
	f := newFixture(t)
	// Each collects 700 € × 11 = 7 700 €: one is under 15 000 €, two are over.
	a, _ := f.svc.Create(validPropertyInput())
	d, _ := f.svc.Detail(a.ID)
	if hasWarning(d.Result, "micro_ceiling") {
		t.Fatalf("one property already warns: %+v", d.Result.Warnings)
	}
	b, _ := f.svc.Create(validPropertyInput())
	reel := validPropertyInput()
	reel.TaxRegime = ReelFoncier
	r, _ := f.svc.Create(reel)

	for _, id := range []string{a.ID, b.ID} {
		d, _ := f.svc.Detail(id)
		w := warning(d.Result, "micro_ceiling")
		if w == nil || !strings.Contains(w.Message, "15 400 €") || !strings.Contains(w.Message, "15 000 €") ||
			!strings.Contains(w.Message, "micro-foncier") {
			t.Errorf("%s: warning = %+v", id, w)
		}
	}
	if d, _ := f.svc.Detail(r.ID); hasWarning(d.Result, "micro_ceiling") {
		t.Errorf("réel property warns: %+v", d.Result.Warnings)
	}
	list, _ := f.svc.List("")
	if !hasWarning(Result{Warnings: list[0].Warnings}, "micro_ceiling") {
		t.Errorf("list doesn't carry the warning: %+v", list[0])
	}
}

func TestMicroBICCeiling(t *testing.T) {
	f := newFixture(t)
	big := validPropertyInput()
	big.TaxRegime = MicroBIC
	big.MonthlyRentCents = 800_000 // 8 000 € × 11 = 88 000 € > 83 600 €

	p, _ := f.svc.Create(big)
	d, _ := f.svc.Detail(p.ID)

	w := warning(d.Result, "micro_ceiling")
	if w == nil || !strings.Contains(w.Message, "micro-BIC") || !strings.Contains(w.Message, "83 600 €") {
		t.Errorf("warning = %+v", w)
	}
}

func TestEvaluateCountsTheUnsavedPropertyAndExcludesItsSavedSelf(t *testing.T) {
	f := newFixture(t)
	saved, _ := f.svc.Create(validPropertyInput())

	// A new property on top of the saved one: 15 400 € > 15 000 €.
	res, _ := f.svc.EvaluateFor("", validPropertyInput())
	if !hasWarning(res, "micro_ceiling") {
		t.Errorf("new property: warnings = %+v", res.Warnings)
	}
	// Re-evaluating the saved one while editing: only itself counts.
	res, _ = f.svc.EvaluateFor(saved.ID, validPropertyInput())
	if hasWarning(res, "micro_ceiling") {
		t.Errorf("editing the only property: warnings = %+v", res.Warnings)
	}
}

func TestDanglingLoanIDStopsStartup(t *testing.T) {
	path := writeData(t, `schema_version: 1
properties:
  - id: 3a7e0c91d24b6f58
    name: T2 Lyon 7e
    price_cents: 15000000
    dpe: D
    monthly_rent_cents: 70000
    monthly_recoverable_charges_cents: 5000
    annual_property_tax_cents: 90000
    annual_condo_fees_cents: 120000
    annual_landlord_insurance_cents: 15000
    agency_managed: false
    tax_regime: micro_foncier
    loan_id: 9f2c4e1a7b3d5f60
    created_at: 2026-11-12T20:41:00+01:00
    updated_at: 2026-11-12T20:41:00+01:00
`)
	store := storage.New()
	loan.RegisterSection(store)
	RegisterSections(store)

	err := store.Load(path)

	var loadErr *storage.LoadError
	if !errors.As(err, &loadErr) || loadErr.Line != 14 || !strings.Contains(err.Error(), "« T2 Lyon 7e »") {
		t.Errorf("err = %v", err)
	}
}

func TestInvalidHandEditedPropertyStopsStartup(t *testing.T) {
	path := writeData(t, "schema_version: 1\nproperties:\n  - id: 3a7e0c91d24b6f58\n    name: T2\n    price_cents: 5\n")
	store := storage.New()
	loan.RegisterSection(store)
	RegisterSections(store)

	err := store.Load(path)

	if err == nil || !strings.Contains(err.Error(), "(ligne 5)") || !strings.Contains(err.Error(), "bien « T2 »") {
		t.Errorf("err = %v", err)
	}
}

func hasWarning(r Result, code string) bool { return warning(r, code) != nil }

func warning(r Result, code string) *Warning {
	for i := range r.Warnings {
		if r.Warnings[i].Code == code {
			return &r.Warnings[i]
		}
	}
	return nil
}

func TestDeletingALoanInUseIsRefusedNamingTheProperties(t *testing.T) {
	f := newFixture(t)
	f.loans.RegisterDeleteGuard(f.svc.LoanDeleteGuard)
	used := saveLoan(t, f, "Prêt utilisé")
	free := saveLoan(t, f, "Prêt libre")
	in := validPropertyInput()
	in.LoanID = used.ID
	_, _ = f.svc.Create(in)

	err := f.loans.Delete(used.ID)

	var inUse *loan.InUseError
	if !errors.As(err, &inUse) || inUse.Message != "Ce prêt est utilisé par le bien « T2 Lyon 7e »" {
		t.Fatalf("err = %v", err)
	}
	if err := f.loans.Delete(free.ID); err != nil {
		t.Errorf("free loan: %v", err)
	}

	in.Name = "Studio"
	_, _ = f.svc.Create(in)
	err = f.loans.Delete(used.ID)
	if !errors.As(err, &inUse) || inUse.Message != "Ce prêt est utilisé par les biens « T2 Lyon 7e » et « Studio »" {
		t.Errorf("two properties: err = %v", err)
	}
}

func TestPropertyListOrDetailNamesThreeOrMoreProperties(t *testing.T) {
	if got := usedByMessage([]string{"A", "B", "C"}); got != "Ce prêt est utilisé par les biens « A », « B » et « C »" {
		t.Errorf("message = %q", got)
	}
}

// The user's case (2026-11-13): « SE appart 1 » with the active loan
// « Conso Immo » (15 000 € at 5 % over 5 years, 283,07 €/month). The loan
// applies exactly as saved: cash-flow after tax = 104,30 − 283,07 € = −178,77 €.
func TestActiveLoanAppliesExactlyAsSaved(t *testing.T) {
	f := newFixture(t)
	conso, err := f.loans.Create(loan.Input{
		Name: "Conso Immo", PrincipalCents: 1_500_000, AnnualRatePct: 5, DurationMonths: 60, FirstPaymentMonth: "2027-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	other := saveLoan(t, f, "Prêt du bien")
	appart := Input{
		Name: "SE appart 1", PriceCents: 3_400_000, DPE: "C", MonthlyRentCents: 36_000,
		MonthlyRecoverableChargesCents: 5_000, AnnualPropertyTaxCents: 65_000, AnnualCondoFeesCents: 70_000,
		TaxRegime: MicroFoncier,
	}
	withOwnLoan := appart
	withOwnLoan.Name = "SE appart 2"
	withOwnLoan.LoanID = other.ID
	_, _ = f.svc.Create(appart)
	_, _ = f.svc.Create(withOwnLoan)

	list, err := f.svc.ListWith(ListOptions{ActiveLoanID: conso.ID})

	if err != nil || len(list) != 2 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	for _, s := range list {
		if s.MonthlyNetIncomeCents != 10_430 {
			t.Fatalf("%s: revenus mensuels après impôts = %d, want 104,30 €", s.Name, s.MonthlyNetIncomeCents)
		}
		if s.LoanID != conso.ID || s.CashFlowAfterTaxCents == nil || *s.CashFlowAfterTaxCents != -17_877 {
			t.Errorf("%s: loan = %s, cash-flow = %v, want −178,77 € with « Conso Immo »", s.Name, s.LoanName, s.CashFlowAfterTaxCents)
		}
	}
}

func TestActiveLoanFinancingUsesTheSavedAmount(t *testing.T) {
	f := newFixture(t)
	active := saveLoan(t, f, "Prêt actif") // 50 000 €, 1 000 € de frais
	p, _ := f.svc.Create(validPropertyInput())

	res, err := f.svc.evaluateWithActiveLoan(p.ID, active.ID)

	if err != nil || res.Financing == nil {
		t.Fatalf("res = %+v, err = %v", res, err)
	}
	// Apport = A 162 750 € + 1 000 € fees − 50 000 € borrowed.
	if res.Financing.OwnContributionCents != 11_375_000 || res.Financing.MonthlyPaymentCents != 28_980 {
		t.Errorf("financing = %+v", *res.Financing)
	}
}

func TestUnknownActiveLoan(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.ListWith(ListOptions{ActiveLoanID: "ffffffffffffffff"}); !errors.Is(err, loan.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestServiceStartsUnfinancedTimelinesNextMonth(t *testing.T) {
	f := newFixture(t) // now = 13 November 2026
	p, _ := f.svc.Create(validPropertyInput())

	d, _ := f.svc.Detail(p.ID)

	if d.Result.Timeline == nil || d.Result.Timeline.StartMonth != "2026-12" {
		t.Errorf("timeline = %+v", d.Result.Timeline)
	}
}
