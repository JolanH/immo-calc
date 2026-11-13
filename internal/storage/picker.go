package storage

import (
	"errors"

	"github.com/ncruces/zenity"
)

// Picker shows native file dialogs. A browser file picker only gives the
// file's content, never its path, so the Go process opens the dialog itself.
type Picker interface {
	// Open asks for an existing data file.
	Open() (string, error)
	// SaveAs asks for a new file name, proposing defaultName.
	SaveAs(defaultName string) (string, error)
}

var (
	// ErrPickerCancelled means the user closed the dialog without choosing.
	ErrPickerCancelled = errors.New("file dialog cancelled")
	// ErrPickerUnavailable means no native dialog can be shown (Linux without
	// zenity or kdialog); the UI then asks for a typed path.
	ErrPickerUnavailable = errors.New("no native file dialog available")
)

// ZenityPicker shows dialogs with github.com/ncruces/zenity: native on
// Windows, zenity/kdialog on Linux.
type ZenityPicker struct{}

var yamlFilter = zenity.FileFilters{{Name: "Fichiers immo-calc (*.yaml)", Patterns: []string{"*.yaml", "*.yml"}, CaseFold: true}}

// Open shows an "Ouvrir" dialog filtered on *.yaml.
func (ZenityPicker) Open() (string, error) {
	path, err := zenity.SelectFile(zenity.Title("Ouvrir un fichier de données immo-calc"), yamlFilter)
	return path, pickerError(err)
}

// SaveAs shows an "Enregistrer sous" dialog filtered on *.yaml.
func (ZenityPicker) SaveAs(defaultName string) (string, error) {
	path, err := zenity.SelectFileSave(zenity.Title("Nouveau fichier de données immo-calc"),
		zenity.Filename(defaultName), yamlFilter)
	return path, pickerError(err)
}

func pickerError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, zenity.ErrCanceled):
		return ErrPickerCancelled
	default:
		// Missing zenity/kdialog binaries, no display, ...: fall back to a typed path.
		return errors.Join(ErrPickerUnavailable, err)
	}
}
