package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/tracelog"
	slogjournal "github.com/systemd/slog-journal"
)

// parseSlogLevel converts a string level name to slog.Level.
// Supports: debug, info, warn, error, crit (mapped to error), none (discard).
func parseSlogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error", "crit":
		return slog.LevelError, nil
	case "none":
		return slog.Level(9999), nil
	default:
		return 0, fmt.Errorf("bad log level: %q", level)
	}
}

// newSlogHandler creates a slog.Handler based on format and level strings.
func newSlogHandler(format, level string) (slog.Handler, error) {
	lvl, err := parseSlogLevel(level)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "text":
		return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}), nil
	case "journal":
		return slogjournal.NewHandler(&slogjournal.Options{
			Level:        lvl,
			ReplaceAttr:  journalReplaceAttr,
			ReplaceGroup: normalizeJournalKey,
		})
	default:
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}), nil
	}
}

// normalizeJournalKey converts a slog key or group name to a valid journald
// field name matching ^[A-Z_][A-Z0-9_]*$.
func normalizeJournalKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToUpper(r))
		} else {
			b.WriteByte('_')
		}
	}
	// Strip leading digits/underscores to ensure the key starts with a letter.
	result := strings.TrimLeftFunc(b.String(), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	if result == "" {
		return "UNKNOWN"
	}
	return result
}

func journalReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	a.Key = normalizeJournalKey(a.Key)
	return a
}

// parseTraceLogLevel maps a level string to a pgx tracelog.LogLevel.
func parseTraceLogLevel(level string) tracelog.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return tracelog.LogLevelDebug
	case "info":
		return tracelog.LogLevelInfo
	case "warn":
		return tracelog.LogLevelWarn
	case "error", "crit":
		return tracelog.LogLevelError
	default:
		return tracelog.LogLevelNone
	}
}
