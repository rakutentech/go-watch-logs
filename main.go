package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/jasonlvhit/gocron"
	gmt "github.com/kevincobain2000/go-msteams/src"

	"github.com/rakutentech/go-watch-logs/pkg"
)

type Flags struct {
	filePath     string
	filePathsCap int
	match        string
	ignore       string
	dbPath       string
	post         string

	min              int
	every            uint64
	healthCheckEvery uint64
	proxy            string
	logLevel         int
	msTeamsHook      string
	version          bool
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
	if len(newFilePaths) > f.filePathsCap {
		slog.Error("Too many files found", "count", len(newFilePaths), "cap", f.filePathsCap)
		slog.Info("Capping to", "count", f.filePathsCap)
	}

	cap := f.filePathsCap
	if cap > len(newFilePaths) {
		cap = len(newFilePaths)
	}

	filePaths = newFilePaths[:cap]

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
		if f.healthCheckEvery > 0 {
			if err := gocron.Every(f.healthCheckEvery).Second().Do(sendHealthCheck); err != nil {
				slog.Error("Error scheduling health check", "error", err.Error())
				return
			}
		}
		<-gocron.Start()
	}
}

func cron() {
	filePathsMutex.Lock()
	defer filePathsMutex.Unlock()

	for _, filePath := range filePaths {
		watch(filePath)
	}
	if f.post != "" {
		if _, err := pkg.ExecShell(f.post); err != nil {
			slog.Error("Error running post command", "error", err.Error())
		}
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
	cap := f.filePathsCap
	if cap > len(newFilePaths) {
		cap = len(newFilePaths)
	}

	filePaths = newFilePaths[:cap]

	filePathsMutex.Unlock()
}

func sendHealthCheck() {
	if f.msTeamsHook == "" {
		return
	}
	details := []gmt.Details{
		{
			Label:   "Health Check",
			Message: "All OK, watching logs is running actively next ping in " + fmt.Sprintf("%d", f.healthCheckEvery) + " seconds",
		},
		{
			Label:   "Version",
			Message: version,
		},
		{
			Label:   "File Path Pattern",
			Message: f.filePath,
		},
		{
			Label:   "File Path Cap",
			Message: fmt.Sprintf("%d", f.filePathsCap),
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
			Label:   "Monitoring Every",
			Message: fmt.Sprintf("%d", f.every),
		},
	}
	for idx, filePath := range filePaths {
		details = append(details, gmt.Details{
			Label:   fmt.Sprintf("File Path %d", idx+1),
			Message: filePath,
		})
	}

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.msTeamsHook, f.proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
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
	flag.StringVar(&f.post, "post", "", "run this shell command after every scan")
	flag.Uint64Var(&f.every, "every", 0, "run every n seconds (0 to run once)")
	flag.Uint64Var(&f.healthCheckEvery, "health-check-every", 86400, "run health check every n seconds (0 to disable)")
	flag.IntVar(&f.logLevel, "log-level", 0, "log level (0=info, 1=debug)")
	flag.IntVar(&f.filePathsCap, "file-paths-cap", 100, "max number of file paths to watch")
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
