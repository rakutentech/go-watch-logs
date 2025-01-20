package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
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
	return hex.EncodeToString(bs[:3])
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
func DisplayableStreakNumber(streak int) int {
	l := streak * 2
	if l < 10 {
		return 10
	}
	return l
}

func StreakSymbols(arr []int, length int, minimum int) string {
	var symbols []string
	for _, v := range arr {
		if v >= minimum {
			symbols = append(symbols, "✕")
		} else {
			symbols = append(symbols, "✓")
		}
	}
	// Fill the rest with grey symbols based on streak length
	for i := len(symbols); i < DisplayableStreakNumber(length); i++ {
		symbols = append([]string{"□"}, symbols...)
	}
	// if last is ✕ then replace with ✖(bold)
	if symbols[len(symbols)-1] == "✕" {
		symbols[len(symbols)-1] = "✖"
	}

	return strings.Join(symbols, "")
}

func NumberToK(num int) string {
	if num >= 1000 {
		return strconv.FormatFloat(float64(num)/1000, 'f', 1, 64) + "K"
	}
	return strconv.Itoa(num)
}

// reduce to n lines
func ReduceToNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}
