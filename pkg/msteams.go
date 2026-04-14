package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Details struct {
	Label   string
	Message string
}

type teamsCard struct {
	Type        string            `json:"type"`
	Attachments []teamsAttachment `json:"attachments"`
}

type teamsAttachment struct {
	ContentType string           `json:"contentType"`
	ContentURL  *string          `json:"contentUrl"`
	Content     teamsCardContent `json:"content"`
}

type teamsCardContent struct {
	Schema      string        `json:"$schema"`
	Type        string        `json:"type"`
	Version     string        `json:"version"`
	AccentColor string        `json:"accentColor"`
	Body        []interface{} `json:"body"`
	Actions     []teamsAction `json:"actions,omitempty"`
	MSTeams     teamsMSTeams  `json:"msteams"`
}

type teamsTextBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	ID     string `json:"id,omitempty"`
	Size   string `json:"size,omitempty"`
	Weight string `json:"weight,omitempty"`
	Color  string `json:"color,omitempty"`
}

type teamsFact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type teamsFactSet struct {
	Type  string      `json:"type"`
	Facts []teamsFact `json:"facts"`
	ID    string      `json:"id"`
}

type teamsAction struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type teamsMSTeams struct {
	Width string `json:"width"`
}

func normalizeGitURL(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	u = strings.TrimRight(u, "/")
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return "https://" + u
}

func actionButton(title string, details []Details, gitURL string) []teamsAction {
	if gitURL == "" {
		return nil
	}
	var filePath, match, ignore, lines string
	for _, d := range details {
		switch d.Label {
		case "File":
			filePath = d.Message
		case "Match":
			match = d.Message
		case "Ignore":
			ignore = d.Message
		case "Lines":
			lines = d.Message
		}
	}
	issueBody := fmt.Sprintf("**File:** %s\n**Match:** %s\n**Ignore:** %s\n\n**Lines:**\n```\n%s\n```", filePath, match, ignore, lines)
	q := url.Values{}
	q.Set("title", title)
	q.Set("body", issueBody)
	q.Set("labels", "go-watch-logs")
	normalizedURL := normalizeGitURL(gitURL)
	issueURL := normalizedURL + "/issues/new?" + q.Encode()
	buttonTitle := "Create issue"
	if parsed, err := url.Parse(normalizedURL); err == nil {
		if orgRepo := strings.TrimLeft(parsed.Path, "/"); orgRepo != "" {
			buttonTitle = "Create Issue on " + orgRepo
		}
	}
	return []teamsAction{{
		Type:  "Action.OpenUrl",
		Title: buttonTitle,
		URL:   issueURL,
	}}
}

func sendToTeams(title string, details []Details, gitURL, hookURL string, httpClient *http.Client) error {
	facts := make([]teamsFact, len(details))
	for i, d := range details {
		facts[i] = teamsFact{Title: d.Label, Value: d.Message}
	}

	actions := actionButton(title, details, gitURL)

	card := teamsCard{
		Type: "message",
		Attachments: []teamsAttachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				ContentURL:  nil,
				Content: teamsCardContent{
					Schema:      "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:        "AdaptiveCard",
					Version:     "1.4",
					AccentColor: "bf0000",
					Body: []interface{}{
						teamsTextBlock{
							Type:   "TextBlock",
							Text:   title,
							ID:     "title",
							Size:   "large",
							Weight: "bolder",
							Color:  "accent",
						},
						teamsFactSet{
							Type:  "FactSet",
							Facts: facts,
							ID:    "acFactSet",
						},
					},
					Actions: actions,
					MSTeams: teamsMSTeams{Width: "Full"},
				},
			},
		},
	}

	requestBody, err := json.Marshal(card)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", hookURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := httpClient.Do(req) //nolint:gosec // hookURL is user-configured webhook, not attacker-controlled
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	return err
}

func NotifyOwnErrorToTeams(e error, r slog.Record, msTeamsHook string, httpClient *http.Client) {
	hostname, _ := os.Hostname()
	slog.Info("Sending own error to MS Teams")

	details := []Details{
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
		details = append(details, Details{
			Label:   attr.Key,
			Message: fmt.Sprintf("%v", attr.Value),
		})
		return true
	})

	err := sendToTeams(hostname, details, "", msTeamsHook, httpClient)
	if err != nil {
		// keep it warn to prevent infinite loop from the global handler of slog
		slog.Warn("Error sending to Teams", "error", err.Error())
		return
	}
	slog.Info("Successfully sent own error to MS Teams")
}
