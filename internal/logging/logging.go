// Package logging configure le logger structuré (slog) de l'application.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// ParseLevel convertit un nom de niveau textuel en slog.Level.
// Toute valeur inconnue retombe sur Info.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Setup installe le logger slog par défaut selon le niveau et le format donnés.
// format "json" produit un handler JSON, toute autre valeur un handler texte.
func Setup(level, format string) {
	opts := &slog.HandlerOptions{Level: ParseLevel(level)}
	var h slog.Handler
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}
