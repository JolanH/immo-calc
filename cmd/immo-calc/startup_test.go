package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"immo-calc/internal/storage"
)

func TestOpenStoreUsesTheResolvedFile(t *testing.T) {
	exeDir, userDir := t.TempDir(), t.TempDir()

	st, err := openStore(Config{}, exeDir, userDir, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(exeDir, storage.DefaultDataFile); st.store.Path() != want {
		t.Errorf("data path = %s, want %s", st.store.Path(), want)
	}
	if st.configPath != filepath.Join(exeDir, storage.ConfigFile) {
		t.Errorf("config path = %s", st.configPath)
	}
}

func TestOpenStoreWarnsWhenTheRememberedFileIsGone(t *testing.T) {
	exeDir, userDir := t.TempDir(), t.TempDir()
	gone := filepath.Join(t.TempDir(), "gone.yaml")
	if err := storage.WriteAppConfig(filepath.Join(exeDir, storage.ConfigFile), storage.AppConfig{LastDataFile: gone}); err != nil {
		t.Fatal(err)
	}

	st, err := openStore(Config{}, exeDir, userDir, nil)

	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(st.warning, gone) {
		t.Errorf("warning = %q", st.warning)
	}
}

func TestOpenStoreRefusesAnInvalidFile(t *testing.T) {
	exeDir := t.TempDir()
	data := filepath.Join(exeDir, "bad.yaml")
	if err := os.WriteFile(data, []byte("schema_version: 1\nnope: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := openStore(Config{DataPath: data}, exeDir, t.TempDir(), nil)

	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{data, "(ligne 2)", "Corrigez-le ou déplacez-le, puis relancez immo-calc."} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}
