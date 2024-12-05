package pkg

import (
	"fmt"
	"runtime"

	gmt "github.com/kevincobain2000/go-msteams/src"
)

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

func GetAlertDetails(f *Flags, result *ScanResult) []gmt.Details {
	details := []gmt.Details{
		{
			Label:   "File Path",
			Message: result.FilePath,
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
			Message: fmt.Sprintf("%d", result.ErrorCount),
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
			Label:   "Error Percentage",
			Message: fmt.Sprintf("%.2f", result.ErrorPercent) + "%",
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
