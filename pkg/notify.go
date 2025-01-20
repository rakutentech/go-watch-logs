package pkg

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	gmt "github.com/kevincobain2000/go-msteams/src"
)

func NotifyOwnError(e error, r slog.Record, msTeamsHook, proxy string) {
	slog.Info("Sending own error to MS Teams")
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
	r.Attrs(func(attr slog.Attr) bool {
		details = append(details, gmt.Details{
			Label:   attr.Key,
			Message: fmt.Sprintf("%v", attr.Value),
		})
		return true
	})
	if msTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return
	}
	err := gmt.Send(hostname, details, msTeamsHook, proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent own error to MS Teams")
	}
}

func Notify(result *ScanResult, f Flags, version string) {
	slog.Info("Sending scan results to MS Teams")
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
		{
			Label: "Details",
			Message: fmt.Sprintf(
				"Min Threshold: %d, Lines Read: %d\n\rMatches Found: %d, Ratio %.2f%%",
				f.Min,
				result.LinesRead,
				result.ErrorCount,
				result.ErrorPercent,
			),
		},
		{
			Label:   "Streaks",
			Message: StreakSymbols(result.Streak, f.Streak, f.Min) + "\n\r" + fmt.Sprintf("Last %d failed. Scan counter: %d", f.Streak, result.ScanCount),
		},
	}
	if result.FirstDate != "" || result.LastDate != "" {
		var duration string
		if result.FirstDate != "" && result.LastDate != "" {
			firstDate, err := time.Parse("2006-01-02 15:04:05", result.FirstDate)
			if err != nil {
				duration = "X"
			} else {
				lastDate, err := time.Parse("2006-01-02 15:04:05", result.LastDate)
				if err == nil {
					duration = lastDate.Sub(firstDate).String()
				} else {
					duration = "X"
				}
			}
		}

		details = append(details, gmt.Details{
			Label:   "Range",
			Message: fmt.Sprintf("%s to %s (Duration: %s)", result.FirstDate, result.LastDate, duration),
		})
	}

	var logDetails []interface{} // nolint: prealloc
	for _, detail := range details {
		logDetails = append(logDetails, detail.Label, detail.Message)
	}

	slog.Info("Sending Alert Notify", logDetails...)

	hostname, _ := os.Hostname()

	if f.MSTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return
	}

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
}
