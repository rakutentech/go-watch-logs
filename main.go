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

var f pkg.Flags

var version = "dev"

var filePaths []string
var filePathsMutex sync.Mutex

func main() {
	flags()
	pkg.SetupLoggingStdout(f.LogLevel)
	parseProxy()
	wantsVersion()
	validate()

	var err error
	newFilePaths, err := pkg.FilesByPattern(f.FilePath)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(newFilePaths) == 0 {
		slog.Error("No files found", "filePath", f.FilePath)
		return
	}
	if len(newFilePaths) > f.FilePathsCap {
		slog.Warn("Too many files found", "count", len(newFilePaths), "cap", f.FilePathsCap)
		slog.Info("Capping to", "count", f.FilePathsCap)
	}

	filePaths = pkg.Capped(f.FilePathsCap, newFilePaths)

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
	if f.Every > 0 {
		if err := gocron.Every(f.Every).Second().Do(pkg.PrintMemUsage); err != nil {
			slog.Error("Error scheduling memory usage", "error", err.Error())
			return
		}
		if err := gocron.Every(f.Every).Second().Do(cron); err != nil {
			slog.Error("Error scheduling cron", "error", err.Error())
			return
		}
		if err := gocron.Every(f.Every).Second().Do(syncFilePaths); err != nil {
			slog.Error("Error scheduling syncFilePaths", "error", err.Error())
			return
		}
		if f.HealthCheckEvery > 0 {
			if err := gocron.Every(f.HealthCheckEvery).Second().Do(sendHealthCheck); err != nil {
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
	if f.PostAlways != "" {
		if _, err := pkg.ExecShell(f.PostAlways); err != nil {
			slog.Error("Error running post command", "error", err.Error())
		}
	}
}

func syncFilePaths() {
	var err error
	newFilePaths, err := pkg.FilesByPattern(f.FilePath)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(newFilePaths) == 0 {
		slog.Error("No files found", "filePath", f.FilePath)
		return
	}

	filePathsMutex.Lock()
	filePaths = pkg.Capped(f.FilePathsCap, newFilePaths)

	filePathsMutex.Unlock()
}

func sendHealthCheck() {
	if f.MSTeamsHook == "" {
		return
	}
	details := pkg.GetHealthCheckDetails(&f, version)
	for idx, filePath := range filePaths {
		details = append(details, gmt.Details{
			Label:   fmt.Sprintf("File Path %d", idx+1),
			Message: filePath,
		})
	}

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
}

func validate() {
	if f.FilePath == "" {
		slog.Error("file-path is required")
		os.Exit(1)
	}
}

func watch(filePath string) {
	watcher, err := pkg.NewWatcher(f.DBPath, filePath, f.Match, f.Ignore)
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

	slog.Info("1st line", "line", pkg.Truncate(result.FirstLine, 150))
	slog.Info("Preview line", "line", pkg.Truncate(result.PreviewLine, 150))
	slog.Info("Last line", "line", pkg.Truncate(result.LastLine, 150))

	slog.Info("Scanning complete", "filePath", result.FilePath)

	if result.ErrorCount < 0 {
		return
	}
	if result.ErrorCount < f.Min {
		return
	}
	notify(result)
	if f.PostMin != "" {
		if _, err := pkg.ExecShell(f.PostMin); err != nil {
			slog.Error("Error running post command", "error", err.Error())
		}
	}
}

func notify(result *pkg.ScanResult) {
	if f.MSTeamsHook == "" {
		return
	}

	slog.Info("Sending to MS Teams")
	details := pkg.GetAlertDetails(&f, result)

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
}

func flags() {
	flag.StringVar(&f.FilePath, "file-path", "", "full path to the log file")
	flag.StringVar(&f.DBPath, "db-path", pkg.GetHomedir()+"/.go-watch-logs.db", "path to store db file")
	flag.StringVar(&f.Match, "match", "", "regex for matching errors (empty to match all lines)")
	flag.StringVar(&f.Ignore, "ignore", "", "regex for ignoring errors (empty to ignore none)")
	flag.StringVar(&f.PostAlways, "post-always", "", "run this shell command after every scan")
	flag.StringVar(&f.PostMin, "post-min", "", "run this shell command after every scan when min errors are found")
	flag.Uint64Var(&f.Every, "every", 0, "run every n seconds (0 to run once)")
	flag.Uint64Var(&f.HealthCheckEvery, "health-check-every", 86400, "run health check every n seconds (0 to disable)")
	flag.IntVar(&f.LogLevel, "log-level", 0, "log level (0=info, 1=debug)")
	flag.IntVar(&f.FilePathsCap, "file-paths-cap", 100, "max number of file paths to watch")
	flag.IntVar(&f.Min, "min", 1, "on minimum num of matches, it should notify")
	flag.BoolVar(&f.Version, "version", false, "")

	flag.StringVar(&f.Proxy, "proxy", "", "http proxy for webhooks")
	flag.StringVar(&f.MSTeamsHook, "ms-teams-hook", "", "ms teams webhook")

	flag.Parse()
}

func parseProxy() string {
	systemProxy := pkg.SystemProxy()
	if systemProxy != "" && f.Proxy == "" {
		f.Proxy = systemProxy
	}
	return f.Proxy
}

func wantsVersion() {
	if f.Version {
		slog.Info("Version", "version", version)
		os.Exit(0)
	}
}
