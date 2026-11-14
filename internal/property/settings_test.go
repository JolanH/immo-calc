package property

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"immo-calc/internal/loan"
	"immo-calc/internal/storage"
)

func openStore(t *testing.T, path string) *storage.Store {
	t.Helper()
	store := storage.New()
	loan.RegisterSection(store)
	RegisterSections(store)
	if err := store.Load(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	return store
}

func writeData(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultsApplyWithoutASettingsSection(t *testing.T) {
	svc := NewService(openStore(t, filepath.Join(t.TempDir(), "absent.yaml")), nil)

	got, err := svc.Settings()

	want := Settings{
		MarginalTaxRatePct: 30, SocialChargesUnfurnishedPct: 17.2, SocialChargesFurnishedPct: 18.6,
		NotaryFeePct: 8.5, VacancyMonthsPerYear: 1, AgencyFeePct: 7, AccentColor: "#8b1a1a",
		ThemeMode: "system",
	}
	if err != nil || got != want {
		t.Errorf("settings = %+v, err = %v", got, err)
	}
}

func TestSettingsAreReadFromTheFile(t *testing.T) {
	path := writeData(t, `schema_version: 1
settings:
  marginal_tax_rate_pct: 41
  social_charges_unfurnished_pct: 17.2
  social_charges_furnished_pct: 18.6
  notary_fee_pct: 7.5
  vacancy_months_per_year: 0.5
  agency_fee_pct: 6
`)
	svc := NewService(openStore(t, path), nil)

	got, _ := svc.Settings()

	if got.MarginalTaxRatePct != 41 || got.NotaryFeePct != 7.5 || got.VacancyMonthsPerYear != 0.5 {
		t.Errorf("settings = %+v", got)
	}
}

func TestUpdateSettingsSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	svc := NewService(openStore(t, path), nil)
	s := DefaultSettings()
	s.MarginalTaxRatePct = 11
	s.VacancyMonthsPerYear = 1.5

	if _, err := svc.UpdateSettings(s); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, _ := NewService(openStore(t, path), nil).Settings()
	if got != s {
		t.Errorf("after restart = %+v, want %+v", got, s)
	}
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "social_charges_furnished_pct: 18.6") {
		t.Errorf("file:\n%s", content)
	}
}

func TestSettingsValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Settings)
		field   string
		message string
	}{
		{"TMI not a bracket", func(s *Settings) { s.MarginalTaxRatePct = 25 }, "marginalTaxRatePct", "0, 11, 30, 41 ou 45 %"},
		{"TMI zero", func(s *Settings) { s.MarginalTaxRatePct = 0 }, "", ""},
		{"TMI 45", func(s *Settings) { s.MarginalTaxRatePct = 45 }, "", ""},
		{"negative unfurnished PS", func(s *Settings) { s.SocialChargesUnfurnishedPct = -1 }, "socialChargesUnfurnishedPct", "entre 0 et 30 %"},
		{"furnished PS too high", func(s *Settings) { s.SocialChargesFurnishedPct = 30.1 }, "socialChargesFurnishedPct", "entre 0 et 30 %"},
		{"notary too high", func(s *Settings) { s.NotaryFeePct = 15.5 }, "notaryFeePct", "Les frais de notaire doivent être compris entre 0 et 15 %"},
		{"vacancy too high", func(s *Settings) { s.VacancyMonthsPerYear = 12.5 }, "vacancyMonthsPerYear", "entre 0 et 12 mois"},
		{"vacancy not a half month", func(s *Settings) { s.VacancyMonthsPerYear = 1.25 }, "vacancyMonthsPerYear", "demi-mois"},
		{"vacancy 12", func(s *Settings) { s.VacancyMonthsPerYear = 12 }, "", ""},
		{"negative agency", func(s *Settings) { s.AgencyFeePct = -0.5 }, "agencyFeePct", "Les frais de gestion doivent être compris entre 0 et 15 %"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultSettings()
			tt.mutate(&s)

			errs := ValidateSettings(s)

			if tt.field == "" {
				if len(errs) != 0 {
					t.Errorf("errors = %v", errs)
				}
				return
			}
			if !strings.Contains(errs[tt.field], tt.message) || len(errs) != 1 {
				t.Errorf("errors = %v, want %q on %s", errs, tt.message, tt.field)
			}
		})
	}
}

func TestUpdateSettingsRejectsInvalidValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	svc := NewService(openStore(t, path), nil)
	s := DefaultSettings()
	s.MarginalTaxRatePct = 12

	_, err := svc.UpdateSettings(s)

	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Fields["marginalTaxRatePct"] == "" {
		t.Fatalf("err = %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("the data file was written")
	}
}

func TestHandEditedInvalidSettingsStopStartup(t *testing.T) {
	path := writeData(t, "schema_version: 1\nsettings:\n  marginal_tax_rate_pct: 33\n")
	store := storage.New()
	RegisterSections(store)

	err := store.Load(path)

	var loadErr *storage.LoadError
	if !errors.As(err, &loadErr) || loadErr.Line != 3 || !strings.Contains(err.Error(), "TMI") {
		t.Errorf("err = %v", err)
	}
}

func TestUnknownSettingsKeyStopsStartup(t *testing.T) {
	path := writeData(t, "schema_version: 1\nsettings:\n  social_charges_pct: 17.2\n")
	store := storage.New()
	RegisterSections(store)

	if err := store.Load(path); err == nil || !strings.Contains(err.Error(), "social_charges_pct") {
		t.Errorf("err = %v", err)
	}
}

func settingsRequest(method, body, origin string) *http.Request {
	req := httptest.NewRequest(method, "/api/settings", strings.NewReader(body))
	req.Host = "127.0.0.1:8421"
	req.Header.Set("Origin", origin)
	return req
}

func serveProperty(h *Handlers, req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const localOrigin = "http://127.0.0.1:8421"

func TestSettingsEndpoints(t *testing.T) {
	h := NewHandlers(NewService(openStore(t, filepath.Join(t.TempDir(), "immo-calc.yaml")), nil))

	got := serveProperty(h, settingsRequest(http.MethodGet, "", localOrigin))
	var s Settings
	_ = json.Unmarshal(got.Body.Bytes(), &s)
	if got.Code != http.StatusOK || s != DefaultSettings() {
		t.Fatalf("get: status = %d, body = %s", got.Code, got.Body.String())
	}
	if !strings.Contains(got.Body.String(), `"socialChargesFurnishedPct":18.6`) {
		t.Errorf("JSON shape: %s", got.Body.String())
	}

	body := `{"marginalTaxRatePct":41,"socialChargesUnfurnishedPct":17.2,"socialChargesFurnishedPct":18.6,
		"notaryFeePct":8,"vacancyMonthsPerYear":2,"agencyFeePct":7}`
	put := serveProperty(h, settingsRequest(http.MethodPut, body, localOrigin))
	if put.Code != http.StatusOK || !strings.Contains(put.Body.String(), `"marginalTaxRatePct":41`) {
		t.Errorf("put: status = %d, body = %s", put.Code, put.Body.String())
	}

	bad := serveProperty(h, settingsRequest(http.MethodPut, strings.Replace(body, `41`, `40`, 1), localOrigin))
	if bad.Code != http.StatusBadRequest || !strings.Contains(bad.Body.String(), `"marginalTaxRatePct"`) {
		t.Errorf("invalid: status = %d, body = %s", bad.Code, bad.Body.String())
	}
	foreign := serveProperty(h, settingsRequest(http.MethodPut, body, "http://evil.example"))
	if foreign.Code != http.StatusForbidden {
		t.Errorf("foreign origin: status = %d", foreign.Code)
	}
	malformed := serveProperty(h, settingsRequest(http.MethodPut, `{`, localOrigin))
	if malformed.Code != http.StatusBadRequest {
		t.Errorf("malformed: status = %d", malformed.Code)
	}
}

func TestAccentColorDefaultsToDarkRed(t *testing.T) {
	if DefaultSettings().AccentColor != "#8b1a1a" {
		t.Errorf("default accent = %q", DefaultSettings().AccentColor)
	}
	// A settings section written before the colour existed still loads, with the default.
	path := writeData(t, "schema_version: 1\nsettings:\n  marginal_tax_rate_pct: 41\n  social_charges_unfurnished_pct: 17.2\n"+
		"  social_charges_furnished_pct: 18.6\n  notary_fee_pct: 8.5\n  vacancy_months_per_year: 1\n  agency_fee_pct: 7\n")
	got, err := NewService(openStore(t, path), nil).Settings()
	if err != nil || got.AccentColor != "#8b1a1a" || got.MarginalTaxRatePct != 41 {
		t.Errorf("settings = %+v, err = %v", got, err)
	}
}

func TestAccentColorValidationAndPersistence(t *testing.T) {
	for _, bad := range []string{"red", "#8b1a1", "#8b1a1aff", "8b1a1a", "#gggggg"} {
		s := DefaultSettings()
		s.AccentColor = bad
		if errs := ValidateSettings(s); errs["accentColor"] != "La couleur doit être au format #rrggbb" {
			t.Errorf("%q: errors = %v", bad, errs)
		}
	}
	path := filepath.Join(t.TempDir(), "immo-calc.yaml")
	s := DefaultSettings()
	s.AccentColor = "#1F6FEB"
	saved, err := NewService(openStore(t, path), nil).UpdateSettings(s)
	if err != nil || saved.AccentColor != "#1f6feb" {
		t.Fatalf("saved = %+v, err = %v", saved, err)
	}
	got, _ := NewService(openStore(t, path), nil).Settings()
	if got.AccentColor != "#1f6feb" {
		t.Errorf("after restart = %q", got.AccentColor)
	}
	s.AccentColor = ""
	reset, _ := NewService(openStore(t, path), nil).UpdateSettings(s)
	if reset.AccentColor != "#8b1a1a" {
		t.Errorf("empty colour = %q, want the default", reset.AccentColor)
	}
}

func TestThemeModeDefaultsToSystemAndIsValidated(t *testing.T) {
	if DefaultSettings().ThemeMode != "system" {
		t.Errorf("default theme = %q", DefaultSettings().ThemeMode)
	}
	s := DefaultSettings()
	for _, ok := range []string{"", "system", "light", "dark"} {
		s.ThemeMode = ok
		if errs := ValidateSettings(s); errs["themeMode"] != "" {
			t.Errorf("theme %q rejected: %v", ok, errs)
		}
	}
	s.ThemeMode = "sepia"
	if errs := ValidateSettings(s); errs["themeMode"] == "" {
		t.Error("an unknown theme must be rejected")
	}
	if got := withDefaults(Settings{}); got.ThemeMode != "system" {
		t.Errorf("empty theme = %q, want system", got.ThemeMode)
	}
}
