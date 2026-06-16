package logging

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{" error ", slog.LevelError},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseLevel(tt.in))
		})
	}
}

func TestSetup_DoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() { Setup("debug", "json") })
	assert.NotPanics(t, func() { Setup("info", "text") })
}
