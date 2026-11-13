package main

import (
	"fmt"

	"immo-calc/internal/notify"
)

// reportFatal logs a startup error and shows it to the user in French,
// pointing to the log file.
func reportFatal(show notify.Func, logf func(format string, args ...any), err error, logPath string) {
	logf("fatal: %v", err)
	message := fmt.Sprintf("immo-calc n'a pas pu démarrer :\n%v\n\nDétails dans le journal : %s", err, logPath)
	show("immo-calc", message)
}
