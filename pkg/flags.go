package pkg

import (
	"flag"
)

type Flags struct {
	FilePath       string
	FilePathsCap   int
	FileRecentSecs uint64
	Match          string
	Ignore         string
	PostCommand    string
	LogFile        string

	Min                int
	Streak             int
	Every              uint64
	Proxy              string
	LogLevel           int
	MemLimit           int
	MSTeamsHook        string
	PagerDutyKey       string
	PagerDutyDedupKey  string
	MaxBufferMB        int
	Test               bool
	Version            bool
}

func Parseflags(f *Flags) {
	flag.StringVar(&f.FilePath, "file-path", "", "full path to the file to watch")
	flag.StringVar(&f.FilePath, "f", "", "(short for --file-path) full path to the file to watch")
	flag.StringVar(&f.LogFile, "log-file", "", "full path to output log file. Empty will log to stdout")
	flag.StringVar(&f.Match, "match", "", "regex for matching errors (empty to match all lines)")
	flag.StringVar(&f.Ignore, "ignore", "", "regex for ignoring errors (empty to ignore none)")
	flag.StringVar(&f.PostCommand, "post-cmd", "", "run this shell command after every scan when min errors are found")
	flag.Uint64Var(&f.Every, "every", 0, "run every n seconds (0 to run once)")
	flag.IntVar(&f.LogLevel, "log-level", 0, "log level (0=info, -4=debug, 4=warn, 8=error)")
	flag.IntVar(&f.MemLimit, "mem-limit", 128, "memory limit in MB (0 to disable)")
	flag.IntVar(&f.FilePathsCap, "file-paths-cap", 100, "max number of file paths to watch")
	flag.Uint64Var(&f.FileRecentSecs, "file-recent-secs", 86400, "only files modified in the last n seconds, 0 to disable")
	flag.IntVar(&f.Min, "min", 1, "on minimum num of matches, it should notify")
	flag.IntVar(&f.Streak, "streak", 1, "on minimum num of streak matches, it should notify")
	flag.IntVar(&f.MaxBufferMB, "mbf", 0, "max buffer in MB, default is 0 (not provided) for go's default 64KB")
	flag.BoolVar(&f.Version, "version", false, "")
	flag.BoolVar(&f.Test, "test", false, `Quickly test paths or regex
# will test if the input matches the regex
echo test123 | go-watch-logs --match=123 --test
# will test if the file paths are found and list them
go-watch-logs --file-path=./ssl_access.*log --test
	`)

	flag.StringVar(&f.Proxy, "proxy", "", "http proxy for webhooks")
	flag.StringVar(&f.MSTeamsHook, "ms-teams-hook", "", "ms teams webhook")
	flag.StringVar(&f.PagerDutyKey, "pagerduty-key", "", "pagerduty routing/integration key")
	flag.StringVar(&f.PagerDutyDedupKey, "pagerduty-dedupkey", "", "pagerduty uniq key, for grpuping events")

	flag.Parse()
	ParsePostFlags(f)
}

func ParsePostFlags(f *Flags) {
	if f.LogFile != "" {
		if err := MkdirP(f.LogFile); err != nil {
			panic("Failed to ensure directory for log file: " + err.Error())
		}
	}
}
