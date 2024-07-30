package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jasonlvhit/gocron"
	gmt "github.com/kevincobain2000/go-msteams/src"

	"github.com/rakutentech/go-watch-logs/pkg"
)

type Flags struct {
	filePath string
	match    string
	ignore   string
	dbPath   string

	min         int
	every       uint64
	proxy       string
	logLevel    int
	msTeamsHook string
	version     bool
}

var f Flags

var version = "dev"

func main() {
	flags()
	pkg.SetupLoggingStdout(f.logLevel)
	parseProxy()
	wantsVersion()
	validate()
	slog.Info("Flags",
		"filePath", f.filePath,
		"match", f.match,
		"ignore", f.ignore,
		"dbPath", f.dbPath,
		"min", f.min,
		"every", f.every,
		"version", f.version,
		"loglevel", f.logLevel,
		"proxy", f.proxy,
		"msTeamsHook", f.msTeamsHook,
	)

	filePaths, err := pkg.FilesByPattern(f.filePath)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(filePaths) == 0 {
		slog.Error("No files found", "filePath", f.filePath)
		return
	}
	for _, filePath := range filePaths {
		isText, err := pkg.IsTextFile(filePath)
		if err != nil {
			slog.Error("Error checking if file is text", "error", err.Error(), "filePath", filePath)
			return
		}
		if !isText {
			slog.Error("File is not a text file", "filePath", filePath)
			return
		}
	}

	for _, filePath := range filePaths {
		watch(filePath)
	}
	if f.every > 0 {
		cron(filePaths)
	}
}

func cron(filePaths []string) {
	for _, filePath := range filePaths {
		if err := gocron.Every(f.every).Second().Do(watch, filePath); err != nil {
			slog.Error("Error scheduling watch", "error", err.Error(), "filePath", filePath)
			return
		}
	}
	if err := gocron.Every(f.every).Second().Do(pkg.PrintMemUsage); err != nil {
		slog.Error("Error scheduling memory usage", "error", err.Error())
		return
	}
	<-gocron.Start()
}

func validate() {
	if f.filePath == "" {
		slog.Error("file-path is required")
		os.Exit(1)
	}
}

func watch(filePath string) {
	watcher, err := pkg.NewWatcher(f.dbPath, filePath, f.match, f.ignore)
	if err != nil {
		slog.Error("Error creating watcher", "error", err.Error(), "filePath", filePath)
		return
	}
	defer watcher.Close()

	slog.Info("Scanning file", "filePath", filePath)

	result, err := watcher.Scan()
	if err != nil {
		slog.Error("Error scanning file", "error", err.Error(), "filePath", filePath)
		return
	}
	slog.Info("Error count", "count", result.ErrorCount)

	// first line
	slog.Info("1st line", "line", pkg.Truncate(result.FirstLine, 50))

	// last line
	slog.Info("Last line", "line", pkg.Truncate(result.LastLine, 50))

	if result.ErrorCount < 0 {
		return
	}
	if result.ErrorCount < f.min {
		return
	}
	notify(result.ErrorCount, result.FirstLine, result.LastLine)
}

func notify(errorCount int, firstLine, lastLine string) {
	if f.msTeamsHook != "" {
		teamsMsg := fmt.Sprintf("total errors: %d\n\n", errorCount)
		teamsMsg += fmt.Sprintf("first line: %s\n\n", firstLine)
		teamsMsg += fmt.Sprintf("last line: %s\n\n", lastLine)
		slog.Info("Sending to Teams")

		hostname, _ := os.Hostname()
		subject := fmt.Sprintf("match: %s", f.match)
		subject += "," // nolint: goconst
		subject += fmt.Sprintf("ignore: %s", f.ignore)
		subject += "," // nolint: goconst
		subject += fmt.Sprintf("min error: %d", f.min)
		err := gmt.Send(hostname, f.filePath, subject, teamsMsg, f.msTeamsHook, f.proxy)
		if err != nil {
			slog.Error("Error sending to Teams", "error", err.Error())
		} else {
			slog.Info("Sent to Teams")
			slog.Info("Done")
		}
	}
}

func flags() {
	flag.StringVar(&f.filePath, "file-path", "", "full path to the log file")
	flag.StringVar(&f.dbPath, "db-path", pkg.GetHomedir()+"/.go-watch-logs.db", "path to store db file")
	flag.StringVar(&f.match, "match", "", "regex for matching errors (empty to match all lines)")
	flag.StringVar(&f.ignore, "ignore", "", "regex for ignoring errors (empty to ignore none)")
	flag.Uint64Var(&f.every, "every", 0, "run every n seconds (0 to run once)")
	flag.IntVar(&f.logLevel, "log-level", 0, "log level (0=info, 1=debug)")
	flag.IntVar(&f.min, "min", 1, "on minimum num of matches, it should notify")
	flag.BoolVar(&f.version, "version", false, "")

	flag.StringVar(&f.proxy, "proxy", "", "http proxy for webhooks")
	flag.StringVar(&f.msTeamsHook, "ms-teams-hook", "", "ms teams webhook")

	flag.Parse()
}

func parseProxy() string {
	systemProxy := pkg.SystemProxy()
	if systemProxy != "" && f.proxy == "" {
		f.proxy = systemProxy
	}
	return f.proxy
}
func wantsVersion() {
	if f.version {
		slog.Info("Version", "version", version)
		os.Exit(0)
	}
}
