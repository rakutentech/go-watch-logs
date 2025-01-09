package pkg

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	gmt "github.com/kevincobain2000/go-msteams/src"
)

func NotifyOwnError(e error, f Flags) {
	hostname, _ := os.Hostname()
	details := []gmt.Details{
		{
			Label:   "Hostname",
			Message: hostname,
		},
		{
			Label:   "Error",
			Message: e.Error(),
		},
	}
	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent own error to MS Teams")
	}
}

func Notify(result *ScanResult, f Flags, version string) {
	slog.Info("Sending to MS Teams")
	details := GetAlertDetails(&f, version, result)

	var logDetails []interface{} // nolint: prealloc
	for _, detail := range details {
		logDetails = append(logDetails, detail.Label, detail.Message)
	}

	if f.MSTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return
	}
	slog.Info("Sending Alert Notify", logDetails...)

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
}

func GetPanicDetails(f *Flags, m *runtime.MemStats) []gmt.Details {
	return []gmt.Details{
		{
			Label:   "File Path",
			Message: f.FilePath,
		},
		{
			Label:   "Match Pattern",
			Message: f.Match,
		},
		{
			Label:   "Ignore Pattern",
			Message: f.Ignore,
		},
		{
			Label:   "Mem Limit (MB) Exceeded",
			Message: fmt.Sprintf("%d", f.MemLimit),
		},
		{
			Label:   "Alloc (MB)",
			Message: fmt.Sprintf("%d", BToMb(m.Alloc)),
		},
	}
}
func GetHealthCheckDetails(f *Flags, version string) []gmt.Details {
	return []gmt.Details{
		{
			Label:   "Health Check",
			Message: "All OK, go-watch-logs is running actively.",
		},
		{
			Label:   "Next Ping",
			Message: fmt.Sprintf("%d secs", f.HealthCheckEvery),
		},
		{
			Label:   "Version",
			Message: version,
		},
		{
			Label:   "File Path Pattern",
			Message: f.FilePath,
		},
		{
			Label:   "File Path Cap",
			Message: fmt.Sprintf("%d", f.FilePathsCap),
		},
		{
			Label:   "Match Pattern",
			Message: f.Match,
		},
		{
			Label:   "Ignore Pattern",
			Message: f.Ignore,
		},
		{
			Label:   "Min Errors Threshold",
			Message: fmt.Sprintf("%d", f.Min),
		},
		{
			Label:   "Monitoring Every",
			Message: fmt.Sprintf("%d secs", f.Every),
		},
	}
}

func GetAlertDetails(f *Flags, version string, result *ScanResult) []gmt.Details {
	details := []gmt.Details{
		{
			Label:   "go-watch-log version",
			Message: version,
		},
		{
			Label:   "File Path",
			Message: result.FilePath,
		},
		{
			Label:   "Running Every",
			Message: fmt.Sprintf("%d secs", f.Every),
		},
		{
			Label:   "Match Pattern",
			Message: f.Match,
		},
		{
			Label:   "Ignore Pattern",
			Message: f.Ignore,
		},
		{
			Label:   "Min Errors Threshold",
			Message: fmt.Sprintf("%d", f.Min),
		},
		{
			Label:   "Lines Read",
			Message: fmt.Sprintf("%d", result.LinesRead),
		},
		{
			Label:   "Total Errors Found",
			Message: fmt.Sprintf("%d (%.2f)", result.ErrorCount, result.ErrorPercent) + "%",
		},
		{
			Label:   "First Line",
			Message: Truncate(result.FirstLine, TruncateMax),
		},
		{
			Label:   "Mid Lines",
			Message: result.PreviewLine,
		},
		{
			Label:   "Last Line",
			Message: Truncate(result.LastLine, TruncateMax),
		},
	}
	if result.FirstDate != "" || result.LastDate != "" {
		details = append(details, gmt.Details{
			Label:   "Time Range",
			Message: fmt.Sprintf("%s to %s", result.FirstDate, result.LastDate),
		})
	}
	return details
}
