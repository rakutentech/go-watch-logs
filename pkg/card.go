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
			Label:   "Total Errors Found",
			Message: fmt.Sprintf("%d", result.ErrorCount),
		},
		{
			Label:   "First Line",
			Message: Truncate(result.FirstLine, TRUNCATE_MAX),
		},
		{
			Label:   "Mid Lines",
			Message: result.PreviewLine,
		},
		{
			Label:   "Last Line",
			Message: Truncate(result.LastLine, TRUNCATE_MAX),
		},
	}
	if result.FirstDate != "" || result.LastDate != "" {
		details = append(details, gmt.Details{
			Label:   "From - To",
			Message: fmt.Sprintf("%s - %s", result.FirstDate, result.LastDate),
		})
	}
	return details
}
