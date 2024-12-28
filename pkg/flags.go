package pkg

import (
	"flag"
)

type Flags struct {
	FilePath     string
	FilePathsCap int
	Match        string
	Ignore       string
	DBPath       string
	PostAlways   string
	PostCommand  string
	LogFile      string

	Min               int
	Every             uint64
	HealthCheckEvery  uint64
	Proxy             string
	LogLevel          int
	MemLimit          int
	MSTeamsHook       string
	Anomaly           bool
	AnomalyWindowDays int
	NotifyOnlyRecent  bool
	Test              bool
	Version           bool
}

func Parseflags(f *Flags) {
	flag.StringVar(&f.FilePath, "file-path", "", "full path to the file to watch")
	flag.StringVar(&f.FilePath, "f", "", "(short for --file-path) full path to the file to watch")
	flag.StringVar(&f.LogFile, "log-file", "", "full path to output log file")
	flag.StringVar(&f.DBPath, "db-path", GetHomedir()+"/.go-watch-logs.db", "path to store db file. Note dir must exist prior")
	flag.StringVar(&f.Match, "match", "", "regex for matching errors (empty to match all lines)")
	flag.StringVar(&f.Ignore, "ignore", "", "regex for ignoring errors (empty to ignore none)")
	flag.StringVar(&f.PostAlways, "post-always", "", "run this shell command after every scan")
	flag.StringVar(&f.PostCommand, "post-cmd", "", "run this shell command after every scan when min errors are found")
	flag.Uint64Var(&f.Every, "every", 0, "run every n seconds (0 to run once)")
	flag.Uint64Var(&f.HealthCheckEvery, "health-check-every", 0, `run health check every n seconds (0 to disable)
sends health check ping to ms teams webhook
`)
	flag.IntVar(&f.LogLevel, "log-level", 0, "log level (0=info, -4=debug, 4=warn, 8=error)")
	flag.IntVar(&f.MemLimit, "mem-limit", 100, "memory limit in MB (0 to disable)")
	flag.IntVar(&f.FilePathsCap, "file-paths-cap", 100, "max number of file paths to watch")
	flag.IntVar(&f.Min, "min", 1, "on minimum num of matches, it should notify")
	flag.BoolVar(&f.Anomaly, "anomaly", false, "record and watch for anomalies (keeping this true not notify on normal matching errors, only on anomalies)")
	flag.IntVar(&f.AnomalyWindowDays, "anomaly-window-days", 7, `anomaly window days
keep data in DB for n days, older data will be deleted
`)
	flag.BoolVar(&f.NotifyOnlyRecent, "notify-only-recent", true, "Notify on latest file only by timestamp based on --every")
	flag.BoolVar(&f.Version, "version", false, "")
	flag.BoolVar(&f.Test, "test", false, `Quickly test paths or regex
# will test if the input matches the regex
echo test123 | go-watch-logs --match=123 --test
# will test if the file paths are found and list them
go-watch-logs --file-path=./ssl_access.*log --test
	`)

	flag.StringVar(&f.Proxy, "proxy", "", "http proxy for webhooks")
	flag.StringVar(&f.MSTeamsHook, "ms-teams-hook", "", "ms teams webhook")

	flag.Parse()
	ParsePostFlags(f)
}

func ParsePostFlags(f *Flags) {
	if f.LogFile != "" {
		if err := MkdirP(f.LogFile); err != nil {
			panic("Failed to ensure directory for log file: " + err.Error())
		}
	}

	if f.DBPath != "" {
		if err := MkdirP(f.DBPath); err != nil {
			panic("Failed to ensure directory for DB path: " + err.Error())
		}
	}
}
