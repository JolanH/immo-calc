//go:build smoke

// Smoke test of the real binary: run with `go test -tags smoke ./cmd/immo-calc`
// (make test does). It builds immo-calc, starts it, and talks to it over HTTP.
package main

import (
	"bufio"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSmoke(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "immo-calc")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "--no-browser", "--port", "0", "--data", filepath.Join(dir, "data.yaml"))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	base := waitForURL(t, stdout)

	health := httpGet(t, base+"api/health")
	if !strings.Contains(health, `"app":"immo-calc"`) {
		t.Errorf("health = %q", health)
	}
	if index := httpGet(t, base); !strings.Contains(index, `<div id="app">`) {
		t.Errorf("GET / did not serve the Vue index: %q", index)
	}

	req, _ := http.NewRequest(http.MethodPost, base+"api/shutdown", nil)
	req.Header.Set("Origin", strings.TrimSuffix(base, "/"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shutdown status = %d", resp.StatusCode)
	}

	select {
	case err := <-exited:
		if err != nil {
			t.Errorf("process exited with %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("process still running 10 s after shutdown")
	}
}

var urlInLog = regexp.MustCompile(`url=(http://127\.0\.0\.1:\d+/)`)

// waitForURL reads the startup log until it prints the URL.
func waitForURL(t *testing.T, stdout io.Reader) string {
	t.Helper()
	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if m := urlInLog.FindStringSubmatch(scanner.Text()); m != nil {
				found <- m[1]
				break
			}
		}
		_, _ = io.Copy(io.Discard, stdout)
	}()
	select {
	case url := <-found:
		return url
	case <-time.After(10 * time.Second):
		t.Fatal("no URL logged within 10 s")
		return ""
	}
}

func httpGet(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d", url, resp.StatusCode)
	}
	return string(body)
}
