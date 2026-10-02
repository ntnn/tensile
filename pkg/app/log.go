package app

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
)

// LogOptions configure the slog logger of a binary.
type LogOptions struct {
	// Level is the minimum log level.
	Level slog.Level
	// Format is the log output format, "text" or "json".
	Format string
	// AddSource adds source locations to log records.
	AddSource bool
}

// AddFlags binds the flag-configurable options to fs.
func (o *LogOptions) AddFlags(fs *flag.FlagSet) {
	fs.TextVar(&o.Level, "log-level", slog.LevelInfo, "minimum log level (debug, info, warn, error)")
	fs.StringVar(&o.Format, "log-format", "text", "log output format (text, json)")
	fs.BoolVar(&o.AddSource, "log-source", false, "add source locations to log records")
}

// Logger builds a logger writing to w.
func (o LogOptions) Logger(w io.Writer) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{
		Level:     o.Level,
		AddSource: o.AddSource,
	}

	var handler slog.Handler
	switch o.Format {
	case "", "text":
		handler = slog.NewTextHandler(w, opts)
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		return nil, fmt.Errorf("unknown log format %q", o.Format)
	}

	return slog.New(handler), nil
}
