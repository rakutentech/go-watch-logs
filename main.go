package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

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

var filePaths []string
var filePathsMutex sync.Mutex

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

	var err error
	filePaths, err = pkg.FilesByPattern(f.filePath)
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
		if err := gocron.Every(f.every).Second().Do(pkg.PrintMemUsage); err != nil {
			slog.Error("Error scheduling memory usage", "error", err.Error())
			return
		}
		if err := gocron.Every(f.every).Second().Do(cron); err != nil {
			slog.Error("Error scheduling cron", "error", err.Error())
			return
		}
		if err := gocron.Every(f.every).Second().Do(syncFilePaths); err != nil {
			slog.Error("Error scheduling syncFilePaths", "error", err.Error())
			return
		}
		<-gocron.Start()
	}
}

func cron() {
	filePathsMutex.Lock()
	defer filePathsMutex.Unlock()

	for _, filePath := range filePaths {
		watch(filePath)
		slog.Debug("Sleeping for 0.5 seconds")
		time.Sleep(500 * time.Millisecond)
	}
}

func syncFilePaths() {
	var err error
	newFilePaths, err := pkg.FilesByPattern(f.filePath)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(newFilePaths) == 0 {
		slog.Error("No files found", "filePath", f.filePath)
		return
	}

	filePathsMutex.Lock()
	filePaths = newFilePaths
	filePathsMutex.Unlock()
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

	slog.Info("Scanning complete", "filePath", result.FilePath)

	if result.ErrorCount < 0 {
		return
	}
	if result.ErrorCount < f.min {
		return
	}
	notify(result)
}

func notify(result *pkg.ScanResult) {
	if f.msTeamsHook == "" {
		return
	}

	slog.Info("Sending to MS Teams")
	details := []gmt.Details{
		{
			Label:   "File Path",
			Message: result.FilePath,
		},
		{
			Label:   "Match Pattern",
			Message: f.match,
		},
		{
			Label:   "Ignore Pattern",
			Message: f.ignore,
		},
		{
			Label:   "Min Errors Threshold",
			Message: fmt.Sprintf("%d", f.min),
		},
		{
			Label:   "Total Errors Found",
			Message: fmt.Sprintf("%d", result.ErrorCount),
		},
		{
			Label:   "First Line",
			Message: result.FirstLine,
		},
		{
			Label:   "Last Line",
			Message: result.LastLine,
		},
	}

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.msTeamsHook, f.proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
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
