package storage

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"immo-calc/internal/server"
)

// pathRequest is the optional body of open/new: a typed path, used when no
// native dialog is available.
type pathRequest struct {
	Path string `json:"path"`
}

// choosePath returns the typed path from the body, or asks the picker when
// the body is empty. ok is false when an answer has already been written.
func (h *Handlers) choosePath(w http.ResponseWriter, r *http.Request, ask func(Picker) (string, error)) (path string, ok bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil {
		server.WriteError(w, http.StatusBadRequest, "invalid_path", "Requête illisible", nil)
		return "", false
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		var req pathRequest
		if err := json.Unmarshal(body, &req); err != nil {
			server.WriteError(w, http.StatusBadRequest, "invalid_path", "Requête illisible", nil)
			return "", false
		}
		path = strings.TrimSpace(req.Path)
		if path == "" || !filepath.IsAbs(path) {
			server.WriteError(w, http.StatusBadRequest, "invalid_path",
				"Indiquez le chemin complet du fichier (par exemple /home/moi/immo-calc.yaml ou C:\\Users\\moi\\immo-calc.yaml)",
				map[string]string{"path": "Chemin complet requis"})
			return "", false
		}
		return filepath.Clean(path), true
	}

	if h.cfg.Picker == nil {
		h.writePickerUnavailable(w, nil)
		return "", false
	}
	path, err = ask(h.cfg.Picker)
	switch {
	case errors.Is(err, ErrPickerCancelled):
		w.WriteHeader(http.StatusNoContent)
		return "", false
	case errors.Is(err, ErrPickerUnavailable):
		h.writePickerUnavailable(w, err)
		return "", false
	case err != nil:
		slog.Error("file dialog failed", "err", err)
		h.writePickerUnavailable(w, err)
		return "", false
	}
	return path, true
}

func (h *Handlers) writePickerUnavailable(w http.ResponseWriter, err error) {
	slog.Info("no native file dialog, asking for a typed path", "err", err)
	server.WriteError(w, http.StatusBadRequest, "picker_unavailable",
		"Aucune fenêtre de sélection de fichier n'est disponible : saisissez le chemin du fichier.", nil)
}

func (h *Handlers) open(w http.ResponseWriter, r *http.Request) {
	path, ok := h.choosePath(w, r, func(p Picker) (string, error) { return p.Open() })
	if !ok {
		return
	}
	if err := h.store.SwitchTo(path); err != nil {
		h.writeSwitchError(w, err)
		return
	}
	h.afterSwitch(path)
	server.WriteJSON(w, http.StatusOK, h.currentInfo())
}

func (h *Handlers) writeSwitchError(w http.ResponseWriter, err error) {
	var loadErr *LoadError
	if errors.As(err, &loadErr) || errors.Is(err, os.ErrNotExist) {
		server.WriteError(w, http.StatusBadRequest, "invalid_file", err.Error(), nil)
		return
	}
	WriteSaveError(w, err)
}

// afterSwitch remembers the new active file and clears the startup warning.
func (h *Handlers) afterSwitch(path string) {
	slog.Info("switched data file", "path", path)
	h.mu.Lock()
	h.warning = ""
	h.mu.Unlock()
	if h.cfg.ConfigPath == "" {
		return
	}
	if err := WriteAppConfig(h.cfg.ConfigPath, AppConfig{LastDataFile: path}); err != nil {
		slog.Warn("could not remember the data file", "config", h.cfg.ConfigPath, "err", err)
	}
}

// newFileName is proposed in the "Enregistrer sous" dialog.
const newFileName = "immo-calc-nouveau.yaml"

func (h *Handlers) newFile(w http.ResponseWriter, r *http.Request) {
	path, ok := h.choosePath(w, r, func(p Picker) (string, error) { return p.SaveAs(newFileName) })
	if !ok {
		return
	}
	if filepath.Ext(path) == "" {
		path += ".yaml"
	}
	if err := h.store.CreateNew(path); err != nil {
		var createErr *createError
		switch {
		case errors.Is(err, fs.ErrExist):
			server.WriteError(w, http.StatusBadRequest, "file_exists", err.Error(), map[string]string{"path": "Ce fichier existe déjà"})
		case errors.As(err, &createErr):
			server.WriteError(w, http.StatusBadRequest, "invalid_path", err.Error(), map[string]string{"path": "Impossible de créer ce fichier"})
		default:
			WriteSaveError(w, err)
		}
		return
	}
	h.afterSwitch(path)
	server.WriteJSON(w, http.StatusOK, h.currentInfo())
}
