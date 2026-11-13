//go:build !windows

package notify

import "os/exec"

// Fatal sends a critical desktop notification with notify-send, if it's
// installed. Otherwise it does nothing: the error is in the log file.
func Fatal(title, message string) {
	path, err := exec.LookPath("notify-send")
	if err != nil {
		return
	}
	_ = exec.Command(path, "--urgency=critical", "--app-name=immo-calc", title, message).Run()
}
