package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const (
	// DefaultDataFile is the data file used next to the executable.
	DefaultDataFile = "immo-calc.yaml"
	// ConfigFile remembers the last data file used.
	ConfigFile = "immo-calc.config.yaml"
)

// AppConfig is the small app config file (not the data file).
type AppConfig struct {
	LastDataFile string `yaml:"last_data_file,omitempty"`
}

// Resolution is the data file chosen at startup.
type Resolution struct {
	Path    string
	Warning string // French, shown as a banner; empty when all is well
}

// ResolveDataFile picks the data file, first match wins: the --data flag,
// then the remembered file if it still exists, then the default file next to
// the executable.
func ResolveDataFile(flagPath string, cfg AppConfig, exeDir string, exists func(string) bool) Resolution {
	if flagPath != "" {
		return Resolution{Path: flagPath}
	}
	def := filepath.Join(exeDir, DefaultDataFile)
	if cfg.LastDataFile == "" {
		return Resolution{Path: def}
	}
	if exists(cfg.LastDataFile) {
		return Resolution{Path: cfg.LastDataFile}
	}
	return Resolution{Path: def, Warning: fmt.Sprintf(
		"Le fichier %s est introuvable, immo-calc utilise %s", cfg.LastDataFile, def)}
}

// FileExists reports whether path exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ConfigPath returns where the app config lives: next to the executable when
// that folder is writable, otherwise in <userConfigDir>/immo-calc/.
func ConfigPath(exeDir, userConfigDir string) string {
	if dirWritable(exeDir) {
		return filepath.Join(exeDir, ConfigFile)
	}
	return filepath.Join(userConfigDir, "immo-calc", ConfigFile)
}

func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".immo-calc-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// ReadAppConfig reads the app config; a missing file is an empty config.
func ReadAppConfig(path string) (AppConfig, error) {
	var cfg AppConfig
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read app config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("parse app config %s: %w", path, err)
	}
	return cfg, nil
}

// WriteAppConfig writes the app config atomically, creating its folder.
func WriteAppConfig(path string, cfg AppConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode app config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create app config folder: %w", err)
	}
	return writeAtomic(osFS{}, path, data)
}
