package main

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/jasonlvhit/gocron"
	"github.com/patrickmn/go-cache"

	"github.com/rakutentech/go-watch-logs/pkg"
)

var f pkg.Flags

var version = "dev"

var filePaths []string
var filePathsMutex sync.Mutex

var cacheMutex sync.Mutex
var caches = make(map[string]*cache.Cache)

func main() {
	pkg.Parseflags(&f)
	pkg.SetupLoggingStdout(f) // nolint: errcheck
	parseProxy()
	wantsVersion()
	validate()

	if f.Test {
		pkg.TestIt(f.FilePath, f.Match)
		return
	}

	syncFilePaths()

	for _, filePath := range filePaths {
		watch(filePath)
	}
	if f.Every > 0 {
		startCron()
	}
}

func syncCaches() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	for filePath := range caches {
		found := false
		for _, f := range filePaths {
			if f == filePath {
				found = true
				break
			}
		}
		if !found {
			slog.Info("Deleting cache obj", "filePath", filePath)
			delete(caches, filePath)
		}
	}
	for _, filePath := range filePaths {
		if _, ok := caches[filePath]; ok {
			continue
		}
		slog.Info("Creating cache obj", "filePath", filePath)
		caches[filePath] = cache.New(cache.NoExpiration, cache.NoExpiration)
	}
}

func startCron() {
	if f.LogLevel == pkg.AppLogLevelDebug {
		if err := gocron.Every(1).Second().Do(pkg.PrintMemUsage, &f); err != nil {
			slog.Error("Error scheduling memory usage", "error", err.Error())
			return
		}
	}

	if err := gocron.Every(f.Every).Second().Do(cronWatch); err != nil {
		slog.Error("Error scheduling cron", "error", err.Error())
		return
	}
	<-gocron.Start()
}

func cronWatch() {
	syncFilePaths()
	filePathsMutex.Lock()
	defer filePathsMutex.Unlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	for _, filePath := range filePaths {
		watch(filePath)
	}
}

func syncFilePaths() {
	slog.Info("Syncing files")

	fpCrawled, err := pkg.FilesByPattern(f.FilePath, f.FileRecentSecs)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
		return
	}
	if len(fpCrawled) == 0 {
		slog.Warn("No files found", "filePath", f.FilePath)
		slog.Warn("Keep watching for new files")
		return
	}

	// Filter and cap file paths
	filePathsMutex.Lock()
	defer filePathsMutex.Unlock()

	filePaths = filterTextFiles(pkg.Capped(f.FilePathsCap, fpCrawled))

	syncCaches()
	slog.Info("Files synced", "fileCount", len(filePaths), "cacheCount", len(caches))
}

// filterTextFiles filters file paths to include only text files.
func filterTextFiles(paths []string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if isText, err := pkg.IsTextFile(path); err == nil && isText {
			filtered = append(filtered, path)
		}
	}
	return filtered
}

func validate() {
	if f.Test {
		return
	}
	if f.FilePath == "" {
		slog.Error("file-path is required")
		return
	}
}

func watch(filePath string) {
	watcher, err := pkg.NewWatcher(filePath, f, caches[filePath])

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
	reportResult(result)
	if _, err := pkg.ExecShell(f.PostCommand); err != nil {
		slog.Error("Error running post command", "error", err.Error())
	}
}

func reportResult(result *pkg.ScanResult) {
	slog.Info("File info", "filePath", result.FilePath, "size", result.FileInfo.Size(), "modTime", result.FileInfo.ModTime())
	slog.Info("Lines read", "count", result.LinesRead)
	slog.Info("Scanning complete", "filePath", result.FilePath)
	slog.Info("1st line", "date", result.FirstDate, "line", pkg.Truncate(result.FirstLine, pkg.TruncateMax))
	slog.Info("Preview line", "line", pkg.Truncate(result.PreviewLine, pkg.TruncateMax))
	slog.Info("Last line", "date", result.LastDate, "line", pkg.Truncate(result.LastLine, pkg.TruncateMax))
	slog.Info("Error count", "percent", fmt.Sprintf("%d (%.2f)", result.ErrorCount, result.ErrorPercent)+"%")
	slog.Info("History", "max streak", f.Streak, "current streaks", result.Streak, "symbols", pkg.StreakSymbols(result.Streak, f.Streak, f.Min))
	slog.Info("Scan", "count", result.ScanCount)

	if result.IsFirstScan() {
		slog.Info("First scan, skipping notification")
		return
	}

	if !pkg.NonStreakZero(result.Streak, f.Streak, f.Min) {
		slog.Info("Streak not met", "streak", f.Streak, "streaks", result.Streak)
		return
	}

	if pkg.IsRecentlyModified(result.FileInfo, f.Every) {
		pkg.Notify(result, f, version)
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
