package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testHTTPClient() *http.Client {
	return &http.Client{}
}

const (
	testGitHubOrgRepoURL = "https://github.com/org/repo"
	testLogFilePath      = "/var/log/app.log"
	testLabelMatch       = "Match"
	testLabelFile        = "File"
	testGitHubAB         = "github.com/a/b"
	testGitHubCD         = "github.com/c/d"
	testSorifyRunURL94   = "http://localhost:8000/sorify/runs/94"
)

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/org/repo", testGitHubOrgRepoURL},
		{"http://github.com/org/repo", "http://github.com/org/repo"},
		{testGitHubOrgRepoURL, testGitHubOrgRepoURL},
		{"https://github.com/org/repo/", testGitHubOrgRepoURL},
		{"  github.com/org/repo  ", testGitHubOrgRepoURL},
		{"github.com/org/repo///", testGitHubOrgRepoURL},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeGitURL(tt.input); got != tt.want {
				t.Errorf("normalizeGitURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSendToTeams_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	details := []Details{
		{Label: testLabelFile, Message: testLogFilePath},
		{Label: testLabelMatch, Message: severityError},
	}

	err := sendToTeams("Test Alert", details, "", nil, server.URL, testHTTPClient())
	if err != nil {
		t.Errorf("sendToTeams() unexpected error: %v", err)
	}
}

func TestSendToTeams_RequestBody(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	details := []Details{
		{Label: testLabelFile, Message: testLogFilePath},
		{Label: testLabelMatch, Message: severityError},
	}

	err := sendToTeams("Test Alert", details, "", nil, server.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("sendToTeams() unexpected error: %v", err)
	}

	var card teamsCard
	if err := json.Unmarshal(capturedBody, &card); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	if card.Type != "message" {
		t.Errorf("card.Type = %q, want %q", card.Type, "message")
	}
	if len(card.Attachments) != 1 {
		t.Fatalf("len(card.Attachments) = %d, want 1", len(card.Attachments))
	}
	if card.Attachments[0].Content.AccentColor != "bf0000" {
		t.Errorf("AccentColor = %q, want %q", card.Attachments[0].Content.AccentColor, "bf0000")
	}
}

func TestSendToTeams_ContentTypeHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-type"); ct != "application/json" {
			t.Errorf("Content-type = %q, want %q", ct, "application/json")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_ = sendToTeams("title", []Details{{Label: "k", Message: "v"}}, "", nil, server.URL, testHTTPClient())
}

func TestSendToTeams_WithGitURL(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	details := []Details{
		{Label: testLabelFile, Message: testLogFilePath},
		{Label: testLabelMatch, Message: severityError},
		{Label: "Ignore", Message: ""},
		{Label: "Lines", Message: "line1\nline2"},
	}

	err := sendToTeams("Alert", details, "github.com/org/repo", nil, server.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("sendToTeams() unexpected error: %v", err)
	}

	var card teamsCard
	if err := json.Unmarshal(capturedBody, &card); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	actions := card.Attachments[0].Content.Actions
	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(actions))
	}
	if actions[0].Type != actionTypeOpenURL {
		t.Errorf("action.Type = %q, want %q", actions[0].Type, actionTypeOpenURL)
	}
	if !strings.Contains(actions[0].URL, "github.com/org/repo/issues/new") {
		t.Errorf("action.URL %q does not contain issues/new", actions[0].URL)
	}
	if !strings.Contains(actions[0].Title, "org/repo") {
		t.Errorf("action.Title %q does not contain org/repo", actions[0].Title)
	}
}

func TestSendToTeams_WithoutGitURL_NoActions(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := sendToTeams("Alert", []Details{{Label: "k", Message: "v"}}, "", nil, server.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("sendToTeams() unexpected error: %v", err)
	}

	var card teamsCard
	if err := json.Unmarshal(capturedBody, &card); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}

	if len(card.Attachments[0].Content.Actions) != 0 {
		t.Errorf("expected no actions when gitURL is empty, got %d", len(card.Attachments[0].Content.Actions))
	}
}

func TestSendToTeams_FactsMatchDetails(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	details := []Details{
		{Label: "Severity", Message: severityError},
		{Label: testLabelFile, Message: "/tmp/app.log"},
		{Label: testLabelMatch, Message: "panic"},
	}

	_ = sendToTeams("Alert", details, "", nil, server.URL, testHTTPClient())

	var card teamsCard
	_ = json.Unmarshal(capturedBody, &card)

	body := card.Attachments[0].Content.Body
	// body[0] is the title TextBlock, body[1] is the FactSet
	raw, _ := json.Marshal(body[1])
	var factSet teamsFactSet
	_ = json.Unmarshal(raw, &factSet)

	if len(factSet.Facts) != len(details) {
		t.Errorf("len(facts) = %d, want %d", len(factSet.Facts), len(details))
	}
	for i, d := range details {
		if factSet.Facts[i].Title != d.Label || factSet.Facts[i].Value != d.Message {
			t.Errorf("fact[%d] = {%q, %q}, want {%q, %q}",
				i, factSet.Facts[i].Title, factSet.Facts[i].Value, d.Label, d.Message)
		}
	}
}

func TestSendToTeams_InvalidHookURL(t *testing.T) {
	err := sendToTeams("title", []Details{}, "", nil, "://bad-url", testHTTPClient())
	if err == nil {
		t.Error("expected error for invalid hook URL, got nil")
	}
}

func TestSendToTeams_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// close the connection abruptly to trigger a client error
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	err := sendToTeams("title", []Details{}, "", nil, server.URL, testHTTPClient())
	if err == nil {
		t.Error("expected error when server closes connection, got nil")
	}
}

func TestNotifyOwnErrorToTeams_Success(_ *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var rec slog.Record
	NotifyOwnErrorToTeams(errors.New("something went wrong"), rec, server.URL, testHTTPClient())
}

func TestNotifyOwnErrorToTeams_BadHook(_ *testing.T) {
	// Should log a warning but not panic when the hook URL is invalid
	var rec slog.Record
	NotifyOwnErrorToTeams(errors.New("test error"), rec, "://bad-url", testHTTPClient())
}

func TestSplitURLs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"   ", nil},
		{",", nil},
		{",,", nil},
		{testGitHubAB, []string{testGitHubAB}},
		{"github.com/a/b,github.com/c/d", []string{testGitHubAB, testGitHubCD}},
		{" github.com/a/b , github.com/c/d ", []string{testGitHubAB, testGitHubCD}},
		{"github.com/a/b,, github.com/c/d,", []string{testGitHubAB, testGitHubCD}},
		// a comma inside a URL must be percent-encoded as %2C so it survives splitting
		{"example.com/t?a=1%2C2,example.com/other", []string{"example.com/t?a=1%2C2", "example.com/other"}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitURLs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitURLs(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitURLs(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func sendAndCaptureActions(t *testing.T, gitURL string, sorifyAction *teamsAction) []teamsAction {
	t.Helper()
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	details := []Details{
		{Label: testLabelFile, Message: testLogFilePath},
		{Label: testLabelMatch, Message: severityError},
		{Label: "Ignore", Message: ""},
		{Label: "Lines", Message: "line1\nline2"},
	}
	if err := sendToTeams("Alert", details, gitURL, sorifyAction, server.URL, testHTTPClient()); err != nil {
		t.Fatalf("sendToTeams() unexpected error: %v", err)
	}

	var card teamsCard
	if err := json.Unmarshal(capturedBody, &card); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return card.Attachments[0].Content.Actions
}

func TestSendToTeams_MultipleGitURLs(t *testing.T) {
	actions := sendAndCaptureActions(t, "github.com/a/b,github.com/c/d", nil)
	if len(actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2", len(actions))
	}
	for i, want := range []string{"a/b", "c/d"} {
		if !strings.Contains(actions[i].Title, want) {
			t.Errorf("actions[%d].Title = %q, want to contain %q", i, actions[i].Title, want)
		}
		if !strings.Contains(actions[i].URL, "github.com/"+want+"/issues/new") {
			t.Errorf("actions[%d].URL = %q, want to contain issues/new for %q", i, actions[i].URL, want)
		}
		if actions[i].Type != actionTypeOpenURL {
			t.Errorf("actions[%d].Type = %q, want %q", i, actions[i].Type, actionTypeOpenURL)
		}
		if actions[i].Style != "" {
			t.Errorf("actions[%d].Style = %q, want empty (default) for git URL button", i, actions[i].Style)
		}
	}
}

func TestSendToTeams_WhitespaceInCommaSeparatedGitURLs(t *testing.T) {
	actions := sendAndCaptureActions(t, " github.com/a/b , github.com/c/d ", nil)
	if len(actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2 git buttons", len(actions))
	}
	if !strings.Contains(actions[0].Title, "a/b") {
		t.Errorf("actions[0].Title = %q, want to contain a/b", actions[0].Title)
	}
	if !strings.Contains(actions[1].Title, "c/d") {
		t.Errorf("actions[1].Title = %q, want to contain c/d", actions[1].Title)
	}
}

func TestTriggerSorifyRun_Success202(t *testing.T) {
	runURL := testSorifyRunURL94
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method = %q, want POST", r.Method)
		}
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202
		_, _ = fmt.Fprintf(w, `{"run_id":94,"run_url":%q,"status":"completed","status_url":"http://localhost:8000/sorify/webhooks/whk_x/runs/94/status"}`, runURL)
	}))
	defer server.Close()

	btn, statusCode, err := triggerSorifyRun(server.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("triggerSorifyRun() unexpected error: %v", err)
	}
	if statusCode != http.StatusAccepted {
		t.Errorf("statusCode = %d, want %d", statusCode, http.StatusAccepted)
	}
	if btn.Title != sorifyRunStartedTitle {
		t.Errorf("btn.Title = %q, want %q", btn.Title, sorifyRunStartedTitle)
	}
	if btn.URL != runURL {
		t.Errorf("btn.URL = %q, want %q", btn.URL, runURL)
	}
	if btn.Type != actionTypeOpenURL {
		t.Errorf("btn.Type = %q, want %q", btn.Type, actionTypeOpenURL)
	}
	if btn.Style != actionStylePositive {
		t.Errorf("btn.Style = %q, want %q (green)", btn.Style, actionStylePositive)
	}
}

func TestTriggerSorifyRun_Conflict409(t *testing.T) {
	runURL := "http://localhost:8000/sorify/runs/97"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusConflict) // 409
		_, _ = fmt.Fprintf(w, `{"message":"A run triggered via this webhook is already in progress.","run_id":97,"run_url":%q,"status_url":"http://localhost:8000/sorify/webhooks/whk_x/runs/97/status"}`, runURL)
	}))
	defer server.Close()

	btn, statusCode, err := triggerSorifyRun(server.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("triggerSorifyRun() unexpected error: %v", err)
	}
	if statusCode != http.StatusConflict {
		t.Errorf("statusCode = %d, want %d", statusCode, http.StatusConflict)
	}
	if btn.Title != sorifyRunningTitle {
		t.Errorf("btn.Title = %q, want %q", btn.Title, sorifyRunningTitle)
	}
	if btn.URL != runURL {
		t.Errorf("btn.URL = %q, want %q", btn.URL, runURL)
	}
	if btn.Style != "" {
		t.Errorf("btn.Style = %q, want %q (default/neutral for 409)", btn.Style, "")
	}
}

func TestTriggerSorifyRun_NonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // 500
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer server.Close()

	_, _, err := triggerSorifyRun(server.URL, testHTTPClient())
	if err == nil {
		t.Error("expected error for 500 status, got nil")
	}
}

func TestTriggerSorifyRun_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted) // 202
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	_, _, err := triggerSorifyRun(server.URL, testHTTPClient())
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestTriggerSorifyRun_MissingRunURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202
		_, _ = w.Write([]byte(`{"run_id":1,"status":"completed"}`))
	}))
	defer server.Close()

	_, _, err := triggerSorifyRun(server.URL, testHTTPClient())
	if err == nil {
		t.Error("expected error when run_url is missing, got nil")
	}
}

func TestTriggerSorifyRun_PreservesQueryParams(t *testing.T) {
	var capturedRequestURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestURL = r.URL.String()
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"run_id":1,"run_url":"http://example.com/run/1"}`))
	}))
	defer server.Close()

	triggerURL := server.URL + "/sorify/webhooks/whk_abc/trigger?test_ids=1,2,3"
	_, _, err := triggerSorifyRun(triggerURL, testHTTPClient())
	if err != nil {
		t.Fatalf("triggerSorifyRun() unexpected error: %v", err)
	}
	if !strings.Contains(capturedRequestURL, "test_ids=1,2,3") {
		t.Errorf("captured request URL %q does not contain test_ids=1,2,3 (raw commas must be preserved)", capturedRequestURL)
	}
}

func TestSendToTeams_WithSorifyAction_Success(t *testing.T) {
	runURL := testSorifyRunURL94
	triggerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"run_id":94,"run_url":%q,"status":"completed"}`, runURL)
	}))
	defer triggerServer.Close()

	btn, _, err := triggerSorifyRun(triggerServer.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("triggerSorifyRun() unexpected error: %v", err)
	}

	actions := sendAndCaptureActions(t, "", &btn)
	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1 sorify button", len(actions))
	}
	if actions[0].Title != sorifyRunStartedTitle {
		t.Errorf("actions[0].Title = %q, want %q", actions[0].Title, sorifyRunStartedTitle)
	}
	if actions[0].URL != runURL {
		t.Errorf("actions[0].URL = %q, want %q", actions[0].URL, runURL)
	}
	if actions[0].Style != actionStylePositive {
		t.Errorf("actions[0].Style = %q, want %q", actions[0].Style, actionStylePositive)
	}
}

func TestSendToTeams_WithSorifyAction_Failure(t *testing.T) {
	triggerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer triggerServer.Close()

	// Trigger fails — Notify builds an error button; replicate that here.
	_, _, err := triggerSorifyRun(triggerServer.URL, testHTTPClient())
	if err == nil {
		t.Fatalf("expected trigger error, got nil")
	}
	sorifyAction := &teamsAction{
		Type:  actionTypeOpenURL,
		Title: sorifyTriggerFailedTitle,
		URL:   triggerServer.URL,
		Style: "destructive",
	}

	actions := sendAndCaptureActions(t, "", sorifyAction)
	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1 error button", len(actions))
	}
	if actions[0].Title != sorifyTriggerFailedTitle {
		t.Errorf("actions[0].Title = %q, want %q", actions[0].Title, sorifyTriggerFailedTitle)
	}
	if actions[0].URL != triggerServer.URL {
		t.Errorf("actions[0].URL = %q, want %q", actions[0].URL, triggerServer.URL)
	}
	if actions[0].Style != "destructive" {
		t.Errorf("actions[0].Style = %q, want %q", actions[0].Style, "destructive")
	}
}

func TestSendToTeams_MixedGitAndSorify(t *testing.T) {
	runURL := testSorifyRunURL94
	triggerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"run_id":94,"run_url":%q,"status":"completed"}`, runURL)
	}))
	defer triggerServer.Close()

	btn, _, err := triggerSorifyRun(triggerServer.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("triggerSorifyRun() unexpected error: %v", err)
	}

	actions := sendAndCaptureActions(t, "github.com/org/repo", &btn)
	if len(actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2 (1 git + 1 sorify)", len(actions))
	}
	// git button first
	if !strings.Contains(actions[0].Title, "org/repo") {
		t.Errorf("actions[0].Title = %q, want to contain org/repo", actions[0].Title)
	}
	if !strings.Contains(actions[0].URL, "github.com/org/repo/issues/new") {
		t.Errorf("actions[0].URL = %q, want to contain issues/new", actions[0].URL)
	}
	if actions[0].Style != "" {
		t.Errorf("actions[0].Style = %q, want empty (default) for git button", actions[0].Style)
	}
	// sorify button second
	if actions[1].Title != sorifyRunStartedTitle {
		t.Errorf("actions[1].Title = %q, want %q", actions[1].Title, sorifyRunStartedTitle)
	}
	if actions[1].URL != runURL {
		t.Errorf("actions[1].URL = %q, want %q", actions[1].URL, runURL)
	}
	if actions[1].Style != actionStylePositive {
		t.Errorf("actions[1].Style = %q, want %q", actions[1].Style, actionStylePositive)
	}
}

func TestSendToTeams_SorifyButtonStyleSerialized(t *testing.T) {
	// Verify the "positive" style is actually emitted in the JSON payload for a
	// successful sorify trigger, and that the git button uses the default (no style).
	runURL := testSorifyRunURL94
	triggerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprintf(w, `{"run_id":94,"run_url":%q,"status":"completed"}`, runURL)
	}))
	defer triggerServer.Close()

	btn, _, err := triggerSorifyRun(triggerServer.URL, testHTTPClient())
	if err != nil {
		t.Fatalf("triggerSorifyRun() unexpected error: %v", err)
	}

	var capturedBody []byte
	teamsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer teamsServer.Close()

	details := []Details{{Label: testLabelFile, Message: "/x"}, {Label: testLabelMatch, Message: "e"}}
	if err := sendToTeams("Alert", details, testGitHubAB, &btn, teamsServer.URL, testHTTPClient()); err != nil {
		t.Fatalf("sendToTeams() unexpected error: %v", err)
	}

	raw := string(capturedBody)
	// only the sorify button carries "positive" style (1 total); git button uses default
	if got := strings.Count(raw, `"style":"positive"`); got != 1 {
		t.Errorf("payload has %d positive styles, want 1 (sorify button only)\n%s", got, raw)
	}
}
