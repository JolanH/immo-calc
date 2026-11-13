// Package browser opens a URL in the user's default browser, using only the
// tools the OS already ships (xdg-open on Linux, rundll32 on Windows).
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
)

// runner starts an external command without waiting for the browser to exit.
type runner func(name string, args ...string) error

// Open opens url in the default browser.
func Open(url string) error {
	return open(runtime.GOOS, startDetached, url)
}

func open(goos string, run runner, url string) error {
	var name string
	var args []string
	switch goos {
	case "linux":
		name, args = "xdg-open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("open browser: unsupported OS %q", goos)
	}
	if err := run(name, args...); err != nil {
		return fmt.Errorf("open browser with %s: %w", name, err)
	}
	return nil
}

func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap the child once it exits
	return nil
}
