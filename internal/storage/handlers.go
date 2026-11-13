package storage

import (
	"errors"
	"log/slog"
	"net/http"

	"go.yaml.in/yaml/v3"

	"immo-calc/internal/server"
)

// HandlerConfig configures the /api/data endpoints.
type HandlerConfig struct {
	// CountKeys are the list sections counted in /api/data/info.
	CountKeys []string
	// Warning is the startup banner (remembered file not found), if any.
	Warning string
}

// Handlers serves the /api/data endpoints.
type Handlers struct {
	store *Store
	cfg   HandlerConfig
}

// NewHandlers returns the /api/data handlers for store.
func NewHandlers(store *Store, cfg HandlerConfig) *Handlers {
	return &Handlers{store: store, cfg: cfg}
}

// Register adds the routes to mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/data/info", h.info)
}

type dataInfo struct {
	Path          string         `json:"path"`
	SchemaVersion int            `json:"schemaVersion"`
	Counts        map[string]int `json:"counts"`
	Warning       string         `json:"warning,omitempty"`
}

func (h *Handlers) info(w http.ResponseWriter, _ *http.Request) {
	server.WriteJSON(w, http.StatusOK, h.currentInfo())
}

func (h *Handlers) currentInfo() dataInfo {
	return dataInfo{
		Path:          h.store.Path(),
		SchemaVersion: CurrentVersion,
		Counts:        h.store.counts(h.cfg.CountKeys),
		Warning:       h.cfg.Warning,
	}
}

// counts returns the number of entries of each list section.
func (s *Store) counts(keys []string) map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	counts := make(map[string]int, len(keys))
	for _, key := range keys {
		if node := s.sections[key]; node != nil && node.Kind == yaml.SequenceNode {
			counts[key] = len(node.Content)
		} else {
			counts[key] = 0
		}
	}
	return counts
}

// WriteSaveError answers an API request whose save failed. Every module's
// write handlers use it after storage.Update.
func WriteSaveError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrFileChangedExternally) {
		server.WriteError(w, http.StatusConflict, "file_changed_externally",
			"Le fichier a été modifié en dehors d'immo-calc", nil)
		return
	}
	slog.Error("save failed", "err", err)
	server.WriteError(w, http.StatusInternalServerError, "save_failed",
		"L'enregistrement a échoué, consultez le journal immo-calc.log", nil)
}
