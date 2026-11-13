package main

import (
	"errors"
	"fmt"
	"log/slog"

	"immo-calc/internal/storage"
)

// startupStore is the loaded data store and what the UI needs to know about it.
type startupStore struct {
	store      *storage.Store
	configPath string
	warning    string
}

// openStore resolves the data file, registers the module sections and loads
// it. Any load failure is fatal: the app refuses to start rather than risk
// overwriting a file it can't read.
func openStore(cfg Config, exeDir, userConfigDir string, register func(*storage.Store)) (startupStore, error) {
	configPath := storage.ConfigPath(exeDir, userConfigDir)
	appCfg, err := storage.ReadAppConfig(configPath)
	if err != nil {
		slog.Warn("ignoring unreadable app config", "path", configPath, "err", err)
	}
	res := storage.ResolveDataFile(cfg.DataPath, appCfg, exeDir, storage.FileExists)
	if res.Warning != "" {
		slog.Warn(res.Warning)
	}

	store := storage.New()
	if register != nil {
		register(store)
	}
	if err := store.Load(res.Path); err != nil {
		var loadErr *storage.LoadError
		if errors.As(err, &loadErr) {
			return startupStore{}, fmt.Errorf("%w. Corrigez-le ou déplacez-le, puis relancez immo-calc.", err)
		}
		return startupStore{}, err
	}
	return startupStore{store: store, configPath: configPath, warning: res.Warning}, nil
}
