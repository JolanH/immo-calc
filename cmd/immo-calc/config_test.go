package main

import (
	"io"
	"testing"
)

func TestParseConfig(t *testing.T) {
	// No --data means "not given": storage resolves the default later.
	defaultData := ""

	tests := []struct {
		name string
		args []string
		want Config
	}{
		{"defaults", nil, Config{Port: 8421, DataPath: defaultData}},
		{"port", []string{"--port", "9000"}, Config{Port: 9000, DataPath: defaultData}},
		{"port zero picks a random port", []string{"--port=0"}, Config{Port: 0, DataPath: defaultData}},
		{"data", []string{"--data", "/tmp/famille.yaml"}, Config{Port: 8421, DataPath: "/tmp/famille.yaml"}},
		{"no browser", []string{"--no-browser"}, Config{Port: 8421, DataPath: defaultData, NoBrowser: true}},
		{"debug", []string{"--debug"}, Config{Port: 8421, DataPath: defaultData, Debug: true}},
		{"version", []string{"--version"}, Config{Port: 8421, DataPath: defaultData, ShowVersion: true}},
		{"all together", []string{"--port", "1234", "--data", "d.yaml", "--no-browser", "--debug"},
			Config{Port: 1234, DataPath: "d.yaml", NoBrowser: true, Debug: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfig(tt.args, io.Discard)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseConfigRejectsInvalidInput(t *testing.T) {
	for _, args := range [][]string{
		{"--port", "-1"},
		{"--port", "65536"},
		{"--port", "abc"},
		{"--unknown"},
		{"extra-argument"},
	} {
		t.Run(args[0], func(t *testing.T) {
			if _, err := parseConfig(args, io.Discard); err == nil {
				t.Errorf("parseConfig(%q) returned no error", args)
			}
		})
	}
}
