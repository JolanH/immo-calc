// Package applog sets up immo-calc's log file. The Windows build has no
// console, so the log file is where startup details and errors end up.
package applog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// FileName is the log file's name, in the executable's directory by default.
const FileName = "immo-calc.log"

// Open creates (or truncates) the log file in preferredDir, falling back to
// fallbackDir (the user's temp dir) when preferredDir isn't writable. It
// returns the open file and its path.
func Open(preferredDir, fallbackDir string) (*os.File, string, error) {
	var errs []error
	for _, dir := range []string{preferredDir, fallbackDir} {
		path := filepath.Join(dir, FileName)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err == nil {
			return f, path, nil
		}
		errs = append(errs, err)
	}
	return nil, "", fmt.Errorf("open log file: %w", errors.Join(errs...))
}

// NewLogger returns a text logger writing to w, at debug level when debug is set.
func NewLogger(w io.Writer, debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}
