package middleware

import (
	"io"
	"log/slog"
	"testing"

	"github.com/levmv/mig"
)

func assertEqual(t *testing.T, expected, actual any, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: mismatch\n\texp: %v\n\tgot: %v", msg, expected, actual)
	}
}

func assertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

func setupSilentLogger(m *mig.Mig) (restore func()) {
	originalLogger := m.Logger
	m.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	return func() {
		m.Logger = originalLogger
	}
}
