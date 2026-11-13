//go:build windows

package notify

import (
	"syscall"
	"unsafe"
)

const (
	mbOK        = 0x00000000
	mbIconError = 0x00000010
)

// Fatal shows a blocking native error message box (user32.MessageBoxW).
func Fatal(title, message string) {
	t, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	proc := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	if proc.Find() != nil {
		return
	}
	_, _, _ = proc.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbOK|mbIconError)
}
