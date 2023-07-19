package deploy

import (
	"regexp"
	"strings"

	"golang.org/x/exp/slog"
)

type LogWritter struct {
}

var space = regexp.MustCompile(`\s+`)

func (w LogWritter) Write(b []byte) (n int, err error) {
	logMessage := space.ReplaceAllString(strings.TrimSpace(strings.ToLower(string(b))), " ")
	slog.Info(logMessage, slog.String("source", "external"))
	return len(b), nil
}
