package pkg

import (
	"crypto/sha256"
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

var (
	tg   *timegrinder.TimeGrinder
	once sync.Once
)

func initTimeGrinder() {
	cfg := timegrinder.Config{}
	var err error
	tg, err = timegrinder.NewTimeGrinder(cfg)
	if err != nil {
		panic(err)
	}
}

func SearchDate(input string) string {
	once.Do(initTimeGrinder)

	ts, ok, err := tg.Extract([]byte(input))
	if err != nil || !ok {
		return ""
	}
	return ts.Format("2006-01-02 15:04:05")
}
