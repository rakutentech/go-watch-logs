package pkg

import (
	"log/slog"
	"os"

	"github.com/MatusOllah/slogcolor"
	"github.com/mattn/go-isatty"
)

func SetupLoggingStdout(logLevel int) {
	opts := &slogcolor.Options{
		Level:       slog.Level(logLevel),
		TimeFormat:  "2006-01-02 15:04:05",
		NoColor:     !isatty.IsTerminal(os.Stderr.Fd()),
		SrcFileMode: slogcolor.ShortFile,
	}
	slog.SetDefault(slog.New(slogcolor.NewHandler(os.Stderr, opts)))
}
