package pkg

import (
	"crypto/sha256"
	"log/slog"
	"sync"

	"github.com/gravwell/gravwell/v3/timegrinder"
)

const (
	TruncateMax = 200
)

func Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return string(bs)
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func LimitString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

var (
	tg   *timegrinder.TimeGrinder
	once sync.Once
)

func initTimeGrinder() error {
	cfg := timegrinder.Config{}
	var err error
	tg, err = timegrinder.NewTimeGrinder(cfg)
	if err != nil {
		return err
	}
	return nil
}

func SearchDate(input string) string {
	var initErr error
	once.Do(func() {
		initErr = initTimeGrinder()
	})
	if initErr != nil {
		slog.Error("Error initializing", "timegrinder", initErr)
		return ""
	}

	ts, ok, err := tg.Extract([]byte(input))
	if err != nil || !ok {
		return ""
	}
	return ts.Format("2006-01-02 15:04:05")
}
