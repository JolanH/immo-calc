package applog

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenCreatesTheLogInThePreferredDir(t *testing.T) {
	exeDir, tempDir := t.TempDir(), t.TempDir()

	f, path, err := Open(exeDir, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer f.Close()

	if want := filepath.Join(exeDir, FileName); path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
}

func TestOpenTruncatesThePreviousLog(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("old run\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, path, err := Open(dir, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("new run\n")
	f.Close()

	got, _ := os.ReadFile(path)
	if string(got) != "new run\n" {
		t.Errorf("log = %q, want only the new run", got)
	}
}

func TestOpenFallsBackWhenThePreferredDirIsNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ignores chmod on directories; the fallback code is OS-independent")
	}
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	readOnly, tempDir := t.TempDir(), t.TempDir()
	if err := os.Chmod(readOnly, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o700) })

	f, path, err := Open(readOnly, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer f.Close()

	if want := filepath.Join(tempDir, FileName); path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
}

func TestNewLoggerLevels(t *testing.T) {
	for _, tt := range []struct {
		debug     bool
		wantDebug bool
	}{{false, false}, {true, true}} {
		var buf bytes.Buffer
		logger := NewLogger(&buf, tt.debug)

		logger.Debug("detail")
		logger.Info("started")

		if got := strings.Contains(buf.String(), "detail"); got != tt.wantDebug {
			t.Errorf("debug=%v: debug line present = %v", tt.debug, got)
		}
		if !strings.Contains(buf.String(), "started") {
			t.Errorf("debug=%v: info line missing", tt.debug)
		}
	}
}
