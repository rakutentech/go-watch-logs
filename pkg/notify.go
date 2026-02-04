package pkg

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	gmt "github.com/kevincobain2000/go-msteams/src"
)

func NotifyOwnErrorToTeams(e error, r slog.Record, msTeamsHook, proxy string) {
	hostname, _ := os.Hostname()
	slog.Info("Sending own error to MS Teams")

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

	err := gmt.Send(hostname, details, msTeamsHook, proxy)
	if err != nil {
		// keep it warn to prevent infinite loop from the global handler of slog
		slog.Warn("Error sending to Teams", "error", err.Error())
		return
	}
	slog.Info("Successfully sent own error to MS Teams")
}

func NotifyOwnErrorToPagerDuty(e error, r slog.Record, pagerDutyKey string, httpClient *http.Client) {
	hostname, _ := os.Hostname()
	slog.Info("Sending own error to PagerDuty")

	details := map[string]any{
		"hostname": hostname,
		"error":    e.Error(),
	}
	r.Attrs(func(attr slog.Attr) bool {
		details[attr.Key] = fmt.Sprintf("%v", attr.Value)
		return true
	})
	pd := NewPagerDuty()

	status, err := pd.Send(hostname, details, pagerDutyKey, "error", "", httpClient)
	if err != nil {
		slog.Warn("Error sending to PagerDuty", "error", err.Error())
		return
	}
	slog.Info("Successfully sent own error to PagerDuty", "status", status)
}

func Notify(result *ScanResult, f Flags, version string, httpClient *http.Client) {
	hostname, _ := os.Hostname()

	// Build MS Teams details
	details := []gmt.Details{
		{
			Label:   "go-watch-log version",
			Message: version,
		},
		{
			Label:   "File",
			Message: result.FilePath,
		},
		{
			Label:   "Match",
			Message: f.Match,
		},
		{
			Label:   "Ignore",
			Message: f.Ignore,
		},
		{
			Label: "Lines",
			Message: fmt.Sprintf(
				"%s\n\r%s\n\r%s",
				Truncate(result.FirstLine, TruncateMax),
				ReduceToNLines(result.PreviewLine, 3),
				Truncate(result.LastLine, TruncateMax),
			),
		},
		{
			Label: "Settings",
			Message: fmt.Sprintf(
				"min (%d), every (%d secs), max streak (%d)",
				f.Min,
				f.Every,
				f.Streak,
			),
		},
		{
			Label: "Scan Details",
			Message: fmt.Sprintf(
				"lines read (%s), %.2f%% errors (%s), scans til date (%s)",
				NumberToK(result.LinesRead),
				result.ErrorPercent,
				NumberToK(result.ErrorCount),
				NumberToK(result.ScanCount),
			),
		},
		{
			Label:   "Streaks",
			Message: StreakSymbols(result.Streak, f.Streak, f.Min),
		},
		{
			Label:   "Countries",
			Message: OrderedAsc(result.CountryCounts),
		},
	}


	if result.FirstDate != "" || result.LastDate != "" {
		var duration string
		if result.FirstDate != "" && result.LastDate != "" {
			firstDate, err := time.Parse("2006-01-02 15:04:05", result.FirstDate)
			if err != nil {
				duration = ""
			} else {
				lastDate, err := time.Parse("2006-01-02 15:04:05", result.LastDate)
				if err == nil {
					duration = fmt.Sprintf("(%s)", lastDate.Sub(firstDate).String())
				} else {
					duration = ""
				}
			}
		}

		details = append(details, gmt.Details{
			Label:   "Range",
			Message: fmt.Sprintf("%s to %s %s", result.FirstDate, result.LastDate, duration),
		})
	}

	var logDetails []any // nolint: prealloc
	for _, detail := range details {
		logDetails = append(logDetails, detail.Label, detail.Message)
	}
	slog.Debug("Sending Alert Notify", logDetails...)

	// Send to MS Teams
	if f.MSTeamsHook != "" {
		slog.Info("Sending scan results to MS Teams")
		err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
		if err != nil {
			// keep it warn to prevent infinite loop from the global handler of slog
			slog.Warn("Error sending to Teams", "error", err.Error())
		} else {
			slog.Info("Successfully sent to MS Teams")
		}
	}

	// Send to PagerDuty
	if f.PagerDutyKey != "" {
		slog.Info("Sending scan results to PagerDuty")

		// Convert from gmt.Details to interface map for PagerDuty
		pdetails := make(map[string]any)
		for _, d := range details {
			pdetails[d.Label] = d.Message
		}

		pd := NewPagerDuty()
		status, err := pd.Send(hostname, pdetails, f.PagerDutyKey, result.Severity, f.PagerDutyDedupKey, httpClient)
		if err != nil {
			slog.Warn("Error sending to PagerDuty", "error", err.Error())
		} else {
			slog.Info("Successfully sent to PagerDuty", "status", status)
		}
	}
}
