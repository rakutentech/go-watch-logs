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

const (
	actionTypeOpenURL        = "Action.OpenUrl"
	actionStylePositive      = "positive"
	sorifyRunStartedTitle    = "Sorify run started"
	sorifyRunningTitle       = "Sorify running"
	sorifyTriggerFailedTitle = "Sorify trigger failed"
	sorifyRateLimitedTitle   = "Sorify rate limited"
)

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
	Style string `json:"style,omitempty"`
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

func splitURLs(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func actionButton(title string, details []Details, gitURLs []string) []teamsAction {
	if len(gitURLs) == 0 {
		return nil
	}

	var actions []teamsAction

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

	for _, gitURL := range gitURLs {
		q := url.Values{}
		q.Set("title", title)
		q.Set("body", issueBody)
		q.Set("labels", "go-watch-logs")
		normalizedURL := normalizeGitURL(gitURL)
		issueURL := normalizedURL + "/issues/new?" + q.Encode()
		buttonTitle := "Create issue"
		if parsed, err := url.Parse(normalizedURL); err == nil {
			if orgRepo := strings.TrimLeft(parsed.Path, "/"); orgRepo != "" {
				buttonTitle = "Create issue on " + orgRepo
			}
		}
		actions = append(actions, teamsAction{
			Type:  actionTypeOpenURL,
			Title: buttonTitle,
			URL:   issueURL,
		})
	}

	return actions
}

// sorifyTriggerResponse is the JSON body returned by a Sorify webhook trigger endpoint.
// Both 202 (run started) and 409 (run already in progress) responses include run_url.
type sorifyTriggerResponse struct {
	RunID     int    `json:"run_id"`
	RunURL    string `json:"run_url"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	StatusURL string `json:"status_url,omitempty"`
}

// notifyTeamsForSorify reports whether an MS Teams message should be sent for a
// Sorify trigger outcome. Teams is notified when a run started (any 2xx status)
// or when the trigger failed (any error); "Sorify running" (409) and "Sorify rate
// limited" (429) outcomes are skipped to avoid duplicate notifications.
func notifyTeamsForSorify(statusCode int, err error) bool {
	return err != nil || (statusCode >= 200 && statusCode < 300)
}

// triggerSorifyRun POSTs to the trigger URL and builds an MS Teams button from the response.
//   - 2xx (e.g. 202) -> "Sorify run started" button (positive/green) pointing at run_url
//   - 409 -> "Sorify running" button (default/neutral) pointing at run_url
//   - 429 -> "Sorify rate limited" button (default/neutral) pointing at run_url when
//     present in the response, otherwise at the trigger URL
//
// Returns the HTTP status code, a button, and an error. The error is non-nil for
// any status code other than 2xx/409/429, network failure, malformed body, or
// when run_url is missing. On a network failure the status code is 0.
// The trigger URL is POSTed verbatim (query params like ?test_ids=1,2,3 are
// preserved); run_url from the response is used verbatim.
//
// Note: the only valid AdaptiveCard action styles are "default", "positive", and
// "destructive" — "warning" is not a valid value and causes Teams to reject the
// entire card.
func triggerSorifyRun(triggerURL string, httpClient *http.Client) (teamsAction, int, error) {
	req, err := http.NewRequest("POST", triggerURL, nil)
	if err != nil {
		return teamsAction{}, 0, fmt.Errorf("build sorify trigger request: %w", err)
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := httpClient.Do(req) //nolint:gosec // triggerURL is user-configured, not attacker-controlled
	if err != nil {
		return teamsAction{}, 0, fmt.Errorf("call sorify trigger: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return teamsAction{}, resp.StatusCode, fmt.Errorf("read sorify trigger response: %w", err)
	}

	slog.Info("Sorify trigger response", "statusCode", resp.StatusCode)
	slog.Debug("sorify trigger response body", "body", string(body))

	if resp.StatusCode == http.StatusTooManyRequests { // 429
		btnURL := triggerURL
		var sr sorifyTriggerResponse
		if json.Unmarshal(body, &sr) == nil && sr.RunURL != "" {
			btnURL = sr.RunURL
		}
		return teamsAction{
			Type:  actionTypeOpenURL,
			Title: sorifyRateLimitedTitle,
			URL:   btnURL,
		}, resp.StatusCode, nil
	}

	var sr sorifyTriggerResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return teamsAction{}, resp.StatusCode, fmt.Errorf("parse sorify trigger response (status %d, body %q): %w", resp.StatusCode, string(body), err)
	}

	if sr.RunURL == "" {
		return teamsAction{}, resp.StatusCode, fmt.Errorf("sorify trigger response missing run_url (status %d, body %q)", resp.StatusCode, string(body))
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300: // 2xx, e.g. 202
		return teamsAction{
			Type:  actionTypeOpenURL,
			Title: sorifyRunStartedTitle,
			URL:   sr.RunURL,
			Style: actionStylePositive,
		}, resp.StatusCode, nil
	case resp.StatusCode == http.StatusConflict: // 409
		return teamsAction{
			Type:  actionTypeOpenURL,
			Title: sorifyRunningTitle,
			URL:   sr.RunURL,
		}, resp.StatusCode, nil
	default:
		return teamsAction{}, resp.StatusCode, fmt.Errorf("sorify trigger unexpected status %d (body %q)", resp.StatusCode, string(body))
	}
}

func sendToTeams(title string, details []Details, gitURL string, sorifyAction *teamsAction, hookURL string, httpClient *http.Client) error {
	facts := make([]teamsFact, len(details))
	for i, d := range details {
		facts[i] = teamsFact{Title: d.Label, Value: d.Message}
	}

	actions := actionButton(title, details, splitURLs(gitURL))

	if sorifyAction != nil {
		actions = append(actions, *sorifyAction)
	}

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

	err := sendToTeams(hostname, details, "", nil, msTeamsHook, httpClient)
	if err != nil {
		// keep it warn to prevent infinite loop from the global handler of slog
		slog.Warn("Error sending to Teams", "error", err.Error())
		return
	}
	slog.Info("Successfully sent own error to MS Teams")
}
