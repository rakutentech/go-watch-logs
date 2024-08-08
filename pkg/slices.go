package pkg

import (
	"github.com/gravwell/gravwell/v3/timegrinder"
)

func Capped[T any](cap int, slice []T) []T {
	capped := cap
	if capped > len(slice) {
		capped = len(slice)
	}
	return slice[:capped]
}

func searchDate(input string) string {
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
