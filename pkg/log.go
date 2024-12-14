package pkg

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/MatusOllah/slogcolor"
	"github.com/mattn/go-isatty"
	"github.com/natefinch/lumberjack"
)

func SetupLoggingStdout(logLevel int, logFile string) error {
	opts := &slogcolor.Options{
		Level:       slog.Level(logLevel),
		TimeFormat:  "2006-01-02 15:04:05",
		NoColor:     !isatty.IsTerminal(os.Stderr.Fd()),
		SrcFileMode: slogcolor.ShortFile,
	}
	if logFile == "" {
		slog.SetDefault(slog.New(slogcolor.NewHandler(os.Stderr, opts)))
	}
	slog.SetDefault(slog.New(slogcolor.NewHandler(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1, // megabytes
		MaxBackups: 3,
		MaxAge:     3, // days
		LocalTime:  true,
		Compress:   true,
	}, opts)))
	fmt.Println("logging to file", logFile)
	return nil
}
