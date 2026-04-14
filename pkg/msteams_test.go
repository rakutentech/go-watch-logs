package pkg

import (
	"encoding/json"
	"errors"
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

func TestNormalizeGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"github.com/org/repo", "https://github.com/org/repo"},
		{"http://github.com/org/repo", "http://github.com/org/repo"},
		{"https://github.com/org/repo", "https://github.com/org/repo"},
		{"https://github.com/org/repo/", "https://github.com/org/repo"},
		{"  github.com/org/repo  ", "https://github.com/org/repo"},
		{"github.com/org/repo///", "https://github.com/org/repo"},
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
		{Label: "File", Message: "/var/log/app.log"},
		{Label: "Match", Message: "error"},
	}

	err := sendToTeams("Test Alert", details, "", server.URL, testHTTPClient())
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
		{Label: "File", Message: "/var/log/app.log"},
		{Label: "Match", Message: "error"},
	}

	err := sendToTeams("Test Alert", details, "", server.URL, testHTTPClient())
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

	_ = sendToTeams("title", []Details{{Label: "k", Message: "v"}}, "", server.URL, testHTTPClient())
}

func TestSendToTeams_WithGitURL(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	details := []Details{
		{Label: "File", Message: "/var/log/app.log"},
		{Label: "Match", Message: "error"},
		{Label: "Ignore", Message: ""},
		{Label: "Lines", Message: "line1\nline2"},
	}

	err := sendToTeams("Alert", details, "github.com/org/repo", server.URL, testHTTPClient())
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
	if actions[0].Type != "Action.OpenUrl" {
		t.Errorf("action.Type = %q, want %q", actions[0].Type, "Action.OpenUrl")
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

	err := sendToTeams("Alert", []Details{{Label: "k", Message: "v"}}, "", server.URL, testHTTPClient())
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
		{Label: "Severity", Message: "error"},
		{Label: "File", Message: "/tmp/app.log"},
		{Label: "Match", Message: "panic"},
	}

	_ = sendToTeams("Alert", details, "", server.URL, testHTTPClient())

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
	err := sendToTeams("title", []Details{}, "", "://bad-url", testHTTPClient())
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

	err := sendToTeams("title", []Details{}, "", server.URL, testHTTPClient())
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
