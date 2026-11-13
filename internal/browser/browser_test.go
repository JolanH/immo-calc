package browser

import (
	"errors"
	"reflect"
	"testing"
)

type recorder struct {
	name string
	args []string
	err  error
}

func (r *recorder) run(name string, args ...string) error {
	r.name, r.args = name, args
	return r.err
}

func TestOpenPicksTheCommandForEachOS(t *testing.T) {
	const url = "http://127.0.0.1:8421/"
	tests := []struct {
		goos string
		name string
		args []string
	}{
		{"linux", "xdg-open", []string{url}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", url}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			r := &recorder{}

			if err := open(tt.goos, r.run, url); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.name != tt.name || !reflect.DeepEqual(r.args, tt.args) {
				t.Errorf("ran %s %v, want %s %v", r.name, r.args, tt.name, tt.args)
			}
		})
	}
}

func TestOpenRejectsUnsupportedOS(t *testing.T) {
	r := &recorder{}

	if err := open("plan9", r.run, "http://127.0.0.1/"); err == nil {
		t.Fatal("expected an error")
	}
	if r.name != "" {
		t.Errorf("ran %q on an unsupported OS", r.name)
	}
}

func TestOpenWrapsRunnerErrors(t *testing.T) {
	r := &recorder{err: errors.New("boom")}

	err := open("linux", r.run, "http://127.0.0.1/")

	if !errors.Is(err, r.err) {
		t.Errorf("err = %v, want it to wrap the runner error", err)
	}
}
