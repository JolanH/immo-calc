package main

import (
	"errors"
	"strings"
	"testing"
)

func TestReportFatalNotifiesInFrenchWithTheLogPath(t *testing.T) {
	var title, message string
	notify := func(t, m string) { title, message = t, m }
	var logged string
	logf := func(format string, args ...any) { logged = format }

	reportFatal(notify, logf, errors.New("listen on 127.0.0.1 port 8421: boom"), "/opt/immo/immo-calc.log")

	if title != "immo-calc" {
		t.Errorf("title = %q", title)
	}
	for _, want := range []string{"immo-calc n'a pas pu démarrer", "boom", "/opt/immo/immo-calc.log"} {
		if !strings.Contains(message, want) {
			t.Errorf("message %q does not contain %q", message, want)
		}
	}
	if logged == "" {
		t.Error("fatal error was not logged")
	}
}
