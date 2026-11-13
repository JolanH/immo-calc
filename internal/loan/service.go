package loan

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"immo-calc/internal/storage"
)

// ErrNotFound means no saved loan has the given id.
var ErrNotFound = errors.New("loan not found")

// ValidationError carries French field messages for an invalid input.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return fmt.Sprintf("invalid loan: %v", e.Fields) }

// Summary is a loan as shown in the list.
type Summary struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	PrincipalCents    int64   `json:"principalCents"`
	AnnualRatePct     float64 `json:"annualRatePct"`
	DurationMonths    int     `json:"durationMonths"`
	MonthlyTotalCents int64   `json:"monthlyTotalCents"`
	TotalCostCents    int64   `json:"totalCostCents"`
}

// Detail is a saved loan with its recomputed results.
type Detail struct {
	Loan
	Result Result `json:"result"`
}

// Service manages saved loan simulations in the data file. Only inputs are
// stored; results are recomputed on every read, so a rule fix applies to
// every saved loan.
type Service struct {
	store *storage.Store
	now   func() time.Time
	newID func() string
}

// NewService returns a service over store, whose loans section must have been
// registered with RegisterSection.
func NewService(store *storage.Store) *Service {
	return &Service{store: store, now: time.Now, newID: newRandomID}
}

// newRandomID returns 16 hex characters from crypto/rand.
func newRandomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read never fails on supported platforms
	return hex.EncodeToString(b)
}

func (s *Service) loans() ([]Loan, error) {
	return storage.Get[[]Loan](s.store, SectionKey)
}

// List returns every saved loan with its key results, in creation order.
func (s *Service) List() ([]Summary, error) {
	loans, err := s.loans()
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(loans))
	for _, l := range loans {
		res, err := Compute(l.Params())
		if err != nil {
			return nil, fmt.Errorf("compute loan %s: %w", l.ID, err)
		}
		out = append(out, Summary{
			ID: l.ID, Name: l.Name, PrincipalCents: l.PrincipalCents, AnnualRatePct: l.AnnualRatePct,
			DurationMonths: l.DurationMonths, MonthlyTotalCents: res.MonthlyTotalCents, TotalCostCents: res.TotalCostCents,
		})
	}
	return out, nil
}

// Get returns a saved loan, or ErrNotFound.
func (s *Service) Get(id string) (Loan, error) {
	loans, err := s.loans()
	if err != nil {
		return Loan{}, err
	}
	for _, l := range loans {
		if l.ID == id {
			return l, nil
		}
	}
	return Loan{}, ErrNotFound
}

// Detail returns a saved loan with its results, or ErrNotFound.
func (s *Service) Detail(id string) (Detail, error) {
	l, err := s.Get(id)
	if err != nil {
		return Detail{}, err
	}
	return detailOf(l)
}

func detailOf(l Loan) (Detail, error) {
	res, err := Compute(l.Params())
	if err != nil {
		return Detail{}, fmt.Errorf("compute loan %s: %w", l.ID, err)
	}
	return Detail{Loan: l, Result: res}, nil
}

// checkInput normalises and validates an input.
func checkInput(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	if errs := Validate(in); len(errs) > 0 {
		return in, &ValidationError{Fields: errs}
	}
	return in, nil
}

// Create validates and saves a new loan.
func (s *Service) Create(in Input) (Loan, error) {
	in, err := checkInput(in)
	if err != nil {
		return Loan{}, err
	}
	now := s.now().Truncate(time.Second)
	l := Loan{ID: s.newID(), Input: in, CreatedAt: now, UpdatedAt: now}
	err = storage.Update(s.store, SectionKey, func(loans *[]Loan) error {
		*loans = append(*loans, l)
		return nil
	})
	if err != nil {
		return Loan{}, err
	}
	return l, nil
}

// Update replaces a saved loan's inputs.
func (s *Service) Update(id string, in Input) (Loan, error) {
	in, err := checkInput(in)
	if err != nil {
		return Loan{}, err
	}
	var updated Loan
	err = storage.Update(s.store, SectionKey, func(loans *[]Loan) error {
		for i := range *loans {
			if (*loans)[i].ID == id {
				(*loans)[i].Input = in
				(*loans)[i].UpdatedAt = s.now().Truncate(time.Second)
				updated = (*loans)[i]
				return nil
			}
		}
		return ErrNotFound
	})
	if err != nil {
		return Loan{}, err
	}
	return updated, nil
}
