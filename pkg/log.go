package pkg

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/MatusOllah/slogcolor"
	"github.com/mattn/go-isatty"
	"github.com/natefinch/lumberjack"
)

// GlobalHandler is a custom handler that catches all logs
type GlobalHandler struct {
	next        slog.Handler
	msTeamsHook string
	proxy       string
}

func (h *GlobalHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level.String() == "ERROR" {
		err := fmt.Errorf("global log capture - Level: %s, Message: %s", r.Level.String(), r.Message)
		NotifyOwnError(err, r, h.msTeamsHook, h.proxy)
	}

	return h.next.Handle(ctx, r)
}

func (h *GlobalHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *GlobalHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &GlobalHandler{next: h.next.WithAttrs(attrs)}
}

func (h *GlobalHandler) WithGroup(name string) slog.Handler {
	return &GlobalHandler{next: h.next.WithGroup(name)}
}

func SetupLoggingStdout(f Flags) error {
	opts := &slogcolor.Options{
		Level:       slog.Level(f.LogLevel),
		TimeFormat:  "2006-01-02 15:04:05",
		NoColor:     !isatty.IsTerminal(os.Stderr.Fd()),
		SrcFileMode: slogcolor.ShortFile,
	}

	var handler slog.Handler
	if f.LogFile == "" {
		handler = slogcolor.NewHandler(os.Stdout, opts)
	} else {
		handler = slogcolor.NewHandler(&lumberjack.Logger{
			Filename:   f.LogFile,
			MaxSize:    10, // megabytes
			MaxBackups: 3,
			MaxAge:     3, // days
			LocalTime:  true,
			Compress:   true,
		}, opts)
		fmt.Println("logging to file", f.LogFile)
	}

	// Wrap the handler with the GlobalHandler
	globalHandler := &GlobalHandler{
		next:        handler,
		msTeamsHook: f.MSTeamsHook,
		proxy:       f.Proxy,
	}
	slog.SetDefault(slog.New(globalHandler))
	return nil
}
