package property

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"immo-calc/internal/loan"
	"immo-calc/internal/storage"
)

// ErrNotFound means no saved property has the given id.
var ErrNotFound = errors.New("property not found")

// Summary is a property as shown in the Biens list.
type Summary struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	PriceCents             int64     `json:"priceCents"`
	MonthlyRentCents       int64     `json:"monthlyRentCents"`
	DPE                    string    `json:"dpe"`
	TaxRegime              Regime    `json:"taxRegime"`
	NetAfterTaxYieldPct    float64   `json:"netAfterTaxYieldPct"`
	NetAfterCreditYieldPct float64   `json:"netAfterCreditYieldPct"`
	MonthlyNetIncomeCents  int64     `json:"monthlyNetIncomeCents"`
	CashFlowAfterTaxCents  *int64    `json:"cashFlowAfterTaxCents"` // with a linked loan only
	LoanID                 string    `json:"loanId"`
	LoanName               string    `json:"loanName"`
	Warnings               []Warning `json:"warnings"`
}

// LoanRef names a property's linked loan.
type LoanRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Detail is a saved property with its results and linked loan.
type Detail struct {
	Property
	Result Result   `json:"result"`
	Loan   *LoanRef `json:"loan"`
}

func newRandomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never fails on supported platforms
	return hex.EncodeToString(b)
}

func (s *Service) properties() ([]Property, error) {
	return storage.Get[[]Property](s.store, PropertiesKey)
}

// candidate is a property to evaluate together with the others.
type candidate struct {
	id string
	in Input
}

// evaluateAll evaluates candidates together: the micro-regime ceilings are
// checked against the rent collected by all of them. With an active loan
// (home page), every candidate is financed by that loan exactly as saved,
// whatever its own linked loan.
func (s *Service) evaluateAll(items []candidate, activeLoan *loan.Loan) ([]Result, error) {
	settings, err := s.Settings()
	if err != nil {
		return nil, err
	}
	var activeFigures *LoanFigures
	if activeLoan != nil {
		f, err := FiguresOf(activeLoan.Params())
		if err != nil {
			return nil, fmt.Errorf("active loan %s: %w", activeLoan.ID, err)
		}
		activeFigures = &f
	}
	figures := map[string]*LoanFigures{}
	results := make([]Result, len(items))
	totals := map[Regime]int64{}
	for i, it := range items {
		var lf *LoanFigures
		if activeFigures != nil {
			lf = activeFigures
		} else if cached, ok := figures[it.in.LoanID]; ok {
			lf = cached
		} else {
			if lf, err = s.loanFigures(it.in.LoanID); err != nil {
				return nil, fmt.Errorf("loan of property %s: %w", it.id, err)
			}
			figures[it.in.LoanID] = lf
		}
		params := it.in.Params()
		params.StartMonth = s.startMonth()
		results[i] = Evaluate(params, settings, lf)
		totals[it.in.TaxRegime] += microReceipts(results[i])
	}
	for i, it := range items {
		if w := ceilingWarning(it.in.TaxRegime, totals[it.in.TaxRegime]); w != nil {
			results[i].Warnings = append(results[i].Warnings, *w)
		}
	}
	return results, nil
}

// startMonth is the month after today: where a bien without a loan starts its
// profitability timeline (like a new loan's default first installment).
func (s *Service) startMonth() loan.Month {
	now := s.now()
	return loan.Month{Year: now.Year(), Month: int(now.Month())}.Next()
}

// ceilingWarning warns when the rent collected by all properties of a micro
// regime is above its ceiling.
func ceilingWarning(r Regime, totalCents int64) *Warning {
	var label string
	var ceiling int64
	switch r {
	case MicroFoncier:
		label, ceiling = "micro-foncier", MicroFoncierCeilingCents
	case MicroBIC:
		label, ceiling = "micro-BIC", MicroBICCeilingCents
	default:
		return nil
	}
	if totalCents <= ceiling {
		return nil
	}
	return &Warning{Code: "micro_ceiling", Severity: "warning", Message: fmt.Sprintf(
		"Les recettes (loyers et charges refacturées) de vos biens en %s (%s) dépassent le plafond de %s : le régime réel s'applique à ces revenus.",
		label, wholeEuros(totalCents), wholeEuros(ceiling))}
}

// wholeEuros formats cents as rounded euros, French style: "15 400 €".
func wholeEuros(cents int64) string {
	digits := fmt.Sprint(roundCents(float64(cents) / 100))
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 && digits[i-1] != '-' {
			b.WriteByte(' ')
		}
		b.WriteRune(d)
	}
	return b.String() + " €"
}

// ListOptions narrow or adjust the property list.
type ListOptions struct {
	LoanID       string // only the properties linked to this loan
	ActiveLoanID string // finance every property with this loan, as saved (home page)
}

// List returns every saved property with its key results, optionally only
// those linked to loanID.
func (s *Service) List(loanID string) ([]Summary, error) {
	return s.ListWith(ListOptions{LoanID: loanID})
}

// activeLoan returns the active loan, or nil when none is asked for.
func (s *Service) activeLoan(activeLoanID string) (*loan.Loan, error) {
	if activeLoanID == "" {
		return nil, nil
	}
	if s.loans == nil {
		return nil, loan.ErrNotFound
	}
	l, err := s.loans.Get(activeLoanID)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// evaluateSaved evaluates every saved property.
func (s *Service) evaluateSaved(activeLoan *loan.Loan) ([]Property, []Result, error) {
	props, err := s.properties()
	if err != nil {
		return nil, nil, err
	}
	items := make([]candidate, len(props))
	for i, p := range props {
		items[i] = candidate{p.ID, p.Input}
	}
	results, err := s.evaluateAll(items, activeLoan)
	return props, results, err
}

// evaluateWithActiveLoan evaluates one saved property with the active loan.
func (s *Service) evaluateWithActiveLoan(id, activeLoanID string) (Result, error) {
	activeLoan, err := s.activeLoan(activeLoanID)
	if err != nil {
		return Result{}, err
	}
	props, results, err := s.evaluateSaved(activeLoan)
	if err != nil {
		return Result{}, err
	}
	for i, p := range props {
		if p.ID == id {
			return results[i], nil
		}
	}
	return Result{}, ErrNotFound
}

// ListWith returns the saved properties with their key results.
func (s *Service) ListWith(opts ListOptions) ([]Summary, error) {
	activeLoan, err := s.activeLoan(opts.ActiveLoanID)
	if err != nil {
		return nil, err
	}
	props, results, err := s.evaluateSaved(activeLoan)
	if err != nil {
		return nil, err
	}
	out := []Summary{}
	for i, p := range props {
		if opts.LoanID != "" && p.LoanID != opts.LoanID {
			continue
		}
		r := results[i]
		sum := Summary{
			ID: p.ID, Name: p.Name, PriceCents: p.PriceCents, MonthlyRentCents: p.MonthlyRentCents, DPE: p.DPE, TaxRegime: p.TaxRegime,
			NetAfterTaxYieldPct: r.NetAfterTaxYieldPct, NetAfterCreditYieldPct: r.NetAfterCreditYieldPct,
			MonthlyNetIncomeCents: r.MonthlyNetIncomeCents,
			LoanID:                p.LoanID, Warnings: r.Warnings,
		}
		if r.Financing != nil {
			cf := r.Financing.CashFlowAfterTaxCents
			sum.CashFlowAfterTaxCents = &cf
		}
		if activeLoan != nil {
			sum.LoanID, sum.LoanName = activeLoan.ID, activeLoan.Name
		} else if ref := s.loanRef(p.LoanID); ref != nil {
			sum.LoanName = ref.Name
		}
		out = append(out, sum)
	}
	return out, nil
}

func (s *Service) loanRef(id string) *LoanRef {
	if id == "" || s.loans == nil {
		return nil
	}
	l, err := s.loans.Get(id)
	if err != nil {
		return nil
	}
	return &LoanRef{ID: l.ID, Name: l.Name}
}

// Get returns a saved property, or ErrNotFound.
func (s *Service) Get(id string) (Property, error) {
	props, err := s.properties()
	if err != nil {
		return Property{}, err
	}
	for _, p := range props {
		if p.ID == id {
			return p, nil
		}
	}
	return Property{}, ErrNotFound
}

// Detail returns a saved property with its results, or ErrNotFound.
func (s *Service) Detail(id string) (Detail, error) {
	p, err := s.Get(id)
	if err != nil {
		return Detail{}, err
	}
	res, err := s.EvaluateFor(id, p.Input)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Property: p, Result: res, Loan: s.loanRef(p.LoanID)}, nil
}

// EvaluateFor evaluates an input (name not required) with the current
// settings, its linked loan, and the other saved properties for the
// ceilings. excludeID is the saved version of the property being edited, so
// it isn't counted twice; "" for a new property.
func (s *Service) EvaluateFor(excludeID string, in Input) (Result, error) {
	if errs := ValidateParams(in, s.loans); len(errs) > 0 {
		return Result{}, &ValidationError{Fields: errs}
	}
	props, err := s.properties()
	if err != nil {
		return Result{}, err
	}
	var items []candidate
	for _, p := range props {
		if p.ID != excludeID {
			items = append(items, candidate{p.ID, p.Input})
		}
	}
	items = append(items, candidate{excludeID, in})
	results, err := s.evaluateAll(items, nil)
	if err != nil {
		return Result{}, err
	}
	return results[len(results)-1], nil
}

func (s *Service) checkInput(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	if errs := ValidateInput(in, s.loans); len(errs) > 0 {
		return in, &ValidationError{Fields: errs}
	}
	return in, nil
}

// Create validates and saves a new property.
func (s *Service) Create(in Input) (Property, error) {
	in, err := s.checkInput(in)
	if err != nil {
		return Property{}, err
	}
	now := s.now().Truncate(time.Second)
	p := Property{ID: s.newID(), Input: in, CreatedAt: now, UpdatedAt: now}
	err = storage.Update(s.store, PropertiesKey, func(props *[]Property) error {
		*props = append(*props, p)
		return nil
	})
	if err != nil {
		return Property{}, err
	}
	return p, nil
}

// Update replaces a saved property's inputs.
func (s *Service) Update(id string, in Input) (Property, error) {
	in, err := s.checkInput(in)
	if err != nil {
		return Property{}, err
	}
	var updated Property
	err = storage.Update(s.store, PropertiesKey, func(props *[]Property) error {
		for i := range *props {
			if (*props)[i].ID == id {
				(*props)[i].Input = in
				(*props)[i].UpdatedAt = s.now().Truncate(time.Second)
				updated = (*props)[i]
				return nil
			}
		}
		return ErrNotFound
	})
	if err != nil {
		return Property{}, err
	}
	return updated, nil
}

const copySuffix = " (copie)"

// Duplicate saves a copy of a property named « <nom> (copie) ».
func (s *Service) Duplicate(id string) (Property, error) {
	original, err := s.Get(id)
	if err != nil {
		return Property{}, err
	}
	in := original.Input
	base := in.Name
	if room := 80 - utf8.RuneCountInString(copySuffix); utf8.RuneCountInString(base) > room {
		base = string([]rune(base)[:room])
	}
	in.Name = base + copySuffix
	return s.Create(in)
}

// Delete removes a property; its linked loan is kept.
func (s *Service) Delete(id string) error {
	return storage.Update(s.store, PropertiesKey, func(props *[]Property) error {
		for i := range *props {
			if (*props)[i].ID == id {
				*props = append((*props)[:i], (*props)[i+1:]...)
				return nil
			}
		}
		return ErrNotFound
	})
}

// LoanDeleteGuard refuses to delete a loan that properties use. Register it
// with loan.Service.RegisterDeleteGuard; its message names the properties.
func (s *Service) LoanDeleteGuard(loanID string) error {
	props, err := s.properties()
	if err != nil {
		return err
	}
	var names []string
	for _, p := range props {
		if p.LoanID == loanID {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return errors.New(usedByMessage(names))
}

// usedByMessage is « Ce prêt est utilisé par le bien « A » » or
// « … par les biens « A », « B » et « C » ».
func usedByMessage(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "« " + n + " »"
	}
	if len(quoted) == 1 {
		return "Ce prêt est utilisé par le bien " + quoted[0]
	}
	return "Ce prêt est utilisé par les biens " + strings.Join(quoted[:len(quoted)-1], ", ") + " et " + quoted[len(quoted)-1]
}
