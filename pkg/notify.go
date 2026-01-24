package pkg

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	gmt "github.com/kevincobain2000/go-msteams/src"
	"github.com/PagerDuty/go-pdagent/pkg/eventsapi"
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
		// keep it warn to prevent infinite loop from the global handler of slog
		slog.Warn("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent own error to MS Teams")
	}
}

// NotifyPagerDuty sends an alert to PagerDuty Event Orchestration with the same details as MS Teams
func NotifyPagerDuty(result *ScanResult, f Flags, hostname string, details []gmt.Details) {
	slog.Info("Sending alert to PagerDuty")
	
	// Determine severity based on error count and percentage
	severity := "error"

	// Create summary from the first detail (usually the file path or hostname)
	summary := fmt.Sprintf("Log errors detected: %d errors (%.2f%%)", 
		result.ErrorCount, result.ErrorPercent)
	
	// Find file path from details for summary
	for _, detail := range details {
		if detail.Label == "File" {
			summary = fmt.Sprintf("Log errors detected in %s: %d errors (%.2f%%)", 
				detail.Message, result.ErrorCount, result.ErrorPercent)
			break
		}
	}

	// Convert MS Teams details to PagerDuty custom_details format
	customDetails := make(map[string]interface{})
	for _, detail := range details {
		customDetails[detail.Label] = detail.Message
	}

	// Create dedup key based on file path to prevent duplicate incidents
	dedupKey := fmt.Sprintf("go-watch-logs-%s", result.FilePath)

	// Create PagerDuty EventV2 using the SDK
	event := &eventsapi.EventV2{
		RoutingKey: f.PagerDutyIntegrationKey,
		Action:     "trigger",
		DedupKey:   dedupKey,
		Payload: eventsapi.PayloadV2{
			Summary:       summary,
			Source:        hostname,
			Severity:      severity,
			CustomDetails: customDetails,
		},
	}

	// Create HTTP client with optional proxy
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	if f.Proxy != "" {
		proxyURL, err := url.Parse(f.Proxy)
		if err != nil {
			slog.Warn("Invalid proxy URL for PagerDuty", "proxy", f.Proxy, "error", err.Error())
		} else {
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			client.Transport = transport
		}
	}

	// Send event using SDK with custom HTTP client
	response, err := eventsapi.Enqueue(event, eventsapi.WithHTTPClient(client))
	if err != nil {
		slog.Warn("Error sending to PagerDuty", "error", err.Error())
		return
	}

	if response.Status == "success" {
		slog.Info("Successfully sent alert to PagerDuty", "status", response.Status, "message", response.Message)
	} else {
		slog.Warn("PagerDuty returned non-success status", "status", response.Status, "message", response.Message)
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

	hostname, _ := os.Hostname()

	if f.MSTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return
	}

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		// keep it warn to prevent infinite loop from the global handler of slog
		slog.Warn("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}

	// Send to PagerDuty if integration key is configured
	if f.PagerDutyIntegrationKey != "" {
		NotifyPagerDuty(result, f, hostname, details)
	}
}
