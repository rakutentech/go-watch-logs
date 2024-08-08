package pkg

import (
	"crypto/sha256"

	"github.com/gravwell/gravwell/v3/timegrinder"
)

const (
	TRUNCATE_MAX = 200
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

func SearchDate(input string) string {
	cfg := timegrinder.Config{}
	tg, err := timegrinder.NewTimeGrinder(cfg)
	if err != nil {
		return ""
	}
	ts, ok, err := tg.Extract([]byte(input))
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return ts.String()
}
