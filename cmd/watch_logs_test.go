package main

import (
	"testing"
)

func TestMain(m *testing.M) {
}

func TestSample1(t *testing.T) {
	var w WatchLogs
	regexps := []string{"str"}
	w.GetMatchedRegexp("test", &regexps)
}
