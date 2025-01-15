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
	pkg.Parseflags(&f)
	pkg.SetupLoggingStdout(f) // nolint: errcheck
	flag.VisitAll(func(f *flag.Flag) {
		slog.Info(f.Name, slog.String("value", f.Value.String()))
	})
	parseProxy()
	wantsVersion()
	validate()

	if f.Test {
		pkg.TestIt(f.FilePath, f.Match)
		return
	}

	var err error
	newFilePaths, err := pkg.FilesByPattern(f.FilePath, f.NotifyOnlyRecent)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(newFilePaths) == 0 {
		slog.Warn("No files found", "filePath", f.FilePath)
		slog.Warn("Keep watching for new files")
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
		startCron()
	}
}

func startCron() {
	if err := gocron.Every(1).Second().Do(pkg.PrintMemUsage, &f); err != nil {
		slog.Error("Error scheduling memory usage", "error", err.Error())
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

	if err := gocron.Every(f.Every).Second().Do(cron); err != nil {
		slog.Error("Error scheduling cron", "error", err.Error())
		return
	}
	<-gocron.Start()
}

func cron() {
	filePathsMutex.Lock()
	defer filePathsMutex.Unlock()

	for _, filePath := range filePaths {
		watch(filePath)
	}
}

func syncFilePaths() {
	var err error
	newFilePaths, err := pkg.FilesByPattern(f.FilePath, f.NotifyOnlyRecent)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(newFilePaths) == 0 {
		slog.Warn("No files found", "filePath", f.FilePath)
		slog.Warn("Keep watching for new files")
		return
	}

	filePathsMutex.Lock()
	filePaths = pkg.Capped(f.FilePathsCap, newFilePaths)

	filePathsMutex.Unlock()
}

func sendHealthCheck() {
	details := pkg.GetHealthCheckDetails(&f, version)
	for idx, filePath := range filePaths {
		details = append(details, gmt.Details{
			Label:   fmt.Sprintf("File Path %d", idx+1),
			Message: filePath,
		})
	}

	var logDetails []interface{} // nolint: prealloc
	for _, detail := range details {
		logDetails = append(logDetails, detail.Label, detail.Message)
	}
	slog.Info("Sending Health Check Notify", logDetails...)
	if f.MSTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return
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
	watcher, err := pkg.NewWatcher(filePath, f)

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
	slog.Info("Lines read", "count", result.LinesRead)
	slog.Info("Scanning complete", "filePath", result.FilePath)
	slog.Info("1st line (truncated to 200 chars)", "date", result.FirstDate, "line", pkg.Truncate(result.FirstLine, pkg.TruncateMax))
	slog.Info("Preview line (truncated to 200 chars)", "line", pkg.Truncate(result.PreviewLine, pkg.TruncateMax))
	slog.Info("Last line (truncated to 200 chars)", "date", result.LastDate, "line", pkg.Truncate(result.LastLine, pkg.TruncateMax))
	slog.Info("Error count", "percent", fmt.Sprintf("%d (%.2f)", result.ErrorCount, result.ErrorPercent)+"%")

	if result.ErrorCount < 0 {
		return
	}
	if result.ErrorCount < f.Min {
		return
	}
	if !f.NotifyOnlyRecent {
		pkg.Notify(result, f, version)
	}

	if f.NotifyOnlyRecent && pkg.IsRecentlyModified(result.FileInfo, f.Every) {
		pkg.Notify(result, f, version)
	}
	if f.PostCommand != "" {
		if _, err := pkg.ExecShell(f.PostCommand); err != nil {
			slog.Error("Error running post command", "error", err.Error())
		}
	}
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
