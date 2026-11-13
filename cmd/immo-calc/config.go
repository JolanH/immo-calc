package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
)

const (
	defaultPort     = 8421
	defaultDataFile = "immo-calc.yaml"
)

// Config is the parsed command line.
type Config struct {
	Port        int    // preferred port; 0 means any free port
	DataPath    string // data file, passed on to storage
	NoBrowser   bool
	Debug       bool
	ShowVersion bool
}

// parseConfig parses the command-line arguments (without the program name).
// exeDir is the executable's directory, used for the default data file.
func parseConfig(args []string, exeDir string, stderr io.Writer) (Config, error) {
	var cfg Config
	fs := flag.NewFlagSet("immo-calc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&cfg.Port, "port", defaultPort, "preferred port (0 = any free port)")
	fs.StringVar(&cfg.DataPath, "data", filepath.Join(exeDir, defaultDataFile), "data file path")
	fs.BoolVar(&cfg.NoBrowser, "no-browser", false, "don't open the browser")
	fs.BoolVar(&cfg.Debug, "debug", false, "verbose logs")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("parse flags: unexpected argument %q", fs.Arg(0))
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("parse flags: port %d out of range 0-65535", cfg.Port)
	}
	return cfg, nil
}
