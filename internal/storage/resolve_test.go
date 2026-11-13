package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveDataFile(t *testing.T) {
	exeDir := filepath.FromSlash("/opt/immo")
	def := filepath.Join(exeDir, DefaultDataFile)
	existing := map[string]bool{"/home/me/famille.yaml": true}
	exists := func(p string) bool { return existing[p] }

	tests := []struct {
		name        string
		flagPath    string
		remembered  string
		wantPath    string
		wantWarning bool
	}{
		{"--data wins over everything", "/tmp/other.yaml", "/home/me/famille.yaml", "/tmp/other.yaml", false},
		{"remembered file that exists", "", "/home/me/famille.yaml", "/home/me/famille.yaml", false},
		{"remembered file that disappeared", "", "/home/me/gone.yaml", def, true},
		{"nothing remembered", "", "", def, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDataFile(tt.flagPath, AppConfig{LastDataFile: tt.remembered}, exeDir, exists)

			if got.Path != tt.wantPath {
				t.Errorf("path = %s, want %s", got.Path, tt.wantPath)
			}
			if (got.Warning != "") != tt.wantWarning {
				t.Errorf("warning = %q", got.Warning)
			}
			if tt.wantWarning && (!strings.Contains(got.Warning, tt.remembered) || !strings.Contains(got.Warning, def) ||
				!strings.Contains(got.Warning, "introuvable")) {
				t.Errorf("warning = %q, want it to name both files", got.Warning)
			}
		})
	}
}

func TestConfigPathPrefersTheExecutableDir(t *testing.T) {
	exeDir, userDir := t.TempDir(), t.TempDir()

	if got, want := ConfigPath(exeDir, userDir), filepath.Join(exeDir, ConfigFile); got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
}

func TestConfigPathFallsBackToTheUserConfigDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ignores chmod on directories; the fallback code is OS-independent")
	}
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	exeDir, userDir := t.TempDir(), t.TempDir()
	if err := os.Chmod(exeDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(exeDir, 0o700) })

	want := filepath.Join(userDir, "immo-calc", ConfigFile)
	if got := ConfigPath(exeDir, userDir); got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
	if names := dirEntries(t, exeDir); len(names) != 0 {
		t.Errorf("probe left files behind: %v", names)
	}
}

func TestAppConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", ConfigFile)

	missing, err := ReadAppConfig(path)
	if err != nil || missing != (AppConfig{}) {
		t.Fatalf("missing config = %+v, err = %v", missing, err)
	}

	if err := WriteAppConfig(path, AppConfig{LastDataFile: "/home/me/famille.yaml"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadAppConfig(path)
	if err != nil || got.LastDataFile != "/home/me/famille.yaml" {
		t.Errorf("config = %+v, err = %v", got, err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "last_data_file: /home/me/famille.yaml\n" {
		t.Errorf("content = %q", content)
	}
}

func TestReadAppConfigRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfigFile)
	_ = os.WriteFile(path, []byte("last_data_file: [\n"), 0o644)

	if _, err := ReadAppConfig(path); err == nil {
		t.Error("expected an error")
	}
}
