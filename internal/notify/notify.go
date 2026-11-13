// Package notify shows a fatal error to a user who started immo-calc by
// double-clicking it, so there is no console to read: a native message box on
// Windows, a desktop notification (notify-send) on Linux when available.
package notify

// Func shows title and message to the user. It must not panic.
type Func func(title, message string)
