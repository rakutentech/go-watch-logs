package pkg

import (
	"fmt"

	gmt "github.com/kevincobain2000/go-msteams/src"
)

func GetHealthCheckDetails(f *Flags, version string) []gmt.Details {
	return []gmt.Details{
		{
			Label:   "Health Check",
			Message: "All OK, go-watch-logs is running actively.",
		},
		{
			Label:   "Next Ping",
			Message: fmt.Sprintf("%d seconds", f.HealthCheckEvery),
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
			Message: fmt.Sprintf("%d", f.Every),
		},
	}
}

func GetAlertDetails(f *Flags, result *ScanResult) []gmt.Details {
	return []gmt.Details{
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
}
