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

// TeamsEvent represents the event data for MS Teams
type TeamsEvent struct {
	Hostname string
	Details  []gmt.Details
}

// PDEvent represents the event data for PagerDuty
type PDEvent struct {
	Event    *eventsapi.EventV2
	HTTPClient *http.Client
}

// processLogResult processes scan results and creates events for both MS Teams and PagerDuty
func processLogResult(result *ScanResult, f Flags, version string) (TeamsEvent, PDEvent, error) {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("Failed to get hostname", "error", err.Error())
		hostname = "unknown"
	}
	
	// Build details array (shared between Teams and PagerDuty)
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

	// Create Teams event
	teamsEvent := TeamsEvent{
		Hostname: hostname,
		Details:  details,
	}

	// Create PagerDuty event
	var pdEvent PDEvent
	
	// Determine severity
	severity := "error"

	// Create summary
	summary := fmt.Sprintf("Log errors detected in %s: %d errors (%.2f%%)", 
		result.FilePath, result.ErrorCount, result.ErrorPercent)

	// Convert MS Teams details to PagerDuty custom_details format
	customDetails := make(map[string]interface{})
	for _, detail := range details {
		customDetails[detail.Label] = detail.Message
	}

	// Create dedup key based on file path to prevent duplicate incidents
	dedupKey := fmt.Sprintf("go-watch-logs-%s", result.FilePath)

	// Create PagerDuty EventV2 using the SDK
	pdEvent.Event = &eventsapi.EventV2{
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

	// Create HTTP client with optional proxy for PagerDuty
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
	pdEvent.HTTPClient = client

	// Log details for debugging
	var logDetails []any // nolint: prealloc
	for _, detail := range details {
		logDetails = append(logDetails, detail.Label, detail.Message)
	}
	slog.Debug("Sending Alert Notify", logDetails...)

	return teamsEvent, pdEvent, nil
}

// sendToTeams sends the event to MS Teams
func sendToTeams(event TeamsEvent, msTeamsHook, proxy string) error {
	if msTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return nil
	}

	slog.Info("Sending scan results to MS Teams")
	err := gmt.Send(event.Hostname, event.Details, msTeamsHook, proxy)
	if err != nil {
		// keep it warn to prevent infinite loop from the global handler of slog
		slog.Warn("Error sending to Teams", "error", err.Error())
		return err
	}
	slog.Info("Successfully sent to MS Teams")
	return nil
}

// sendToPagerDuty sends the event to PagerDuty
func sendToPagerDuty(event PDEvent) error {
	if event.Event == nil {
		return nil
	}

	slog.Info("Sending alert to PagerDuty")
	
	// Send event using SDK with custom HTTP client
	response, err := eventsapi.Enqueue(event.Event, eventsapi.WithHTTPClient(event.HTTPClient))
	if err != nil {
		slog.Warn("Error sending to PagerDuty", "error", err.Error())
		return err
	}

	if response.Status == "success" {
		slog.Info("Successfully sent alert to PagerDuty", "status", response.Status, "message", response.Message)
	} else {
		slog.Warn("PagerDuty returned non-success status", "status", response.Status, "message", response.Message)
	}
	return nil
}

func Notify(result *ScanResult, f Flags, version string) {
	// Process log results into events
	teamsEvent, pdEvent, err := processLogResult(result, f, version)
	if err != nil {
		slog.Warn("Error processing log results", "error", err.Error())
		return
	}

	// Send to MS Teams (failure won't prevent PagerDuty from being sent)
	if err := sendToTeams(teamsEvent, f.MSTeamsHook, f.Proxy); err != nil {
		slog.Warn("Failed to send to MS Teams", "error", err.Error())
	}

	// Send to PagerDuty if integration key is configured (independent of Teams result)
	if f.PagerDutyIntegrationKey != "" {
		if err := sendToPagerDuty(pdEvent); err != nil {
			slog.Warn("Failed to send to PagerDuty", "error", err.Error())
		}
	}
}
