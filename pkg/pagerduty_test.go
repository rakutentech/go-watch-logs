package pkg

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// mockHTTPClient is a mock implementation of http.Client for testing
type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// mockTransport implements http.RoundTripper for testing
type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// createMockHTTPClient creates a mock HTTP client with a given response
func createMockHTTPClient(statusCode int, body string, err error) *http.Client {
	return &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			},
			err: err,
		},
	}
}

func TestNewPagerDuty(t *testing.T) {
	pd := NewPagerDuty()
	if pd == nil {
		t.Error("NewPagerDuty() returned nil")
	}
}

func TestPagerDuty_Send_Success(t *testing.T) {
	tests := []struct {
		name       string
		summary    string
		details    map[string]any
		routingKey string
		severity   string
		dedupKey   string
		statusCode int
		body       string
	}{
		{
			name:       "successful event with all fields",
			summary:    "Test Alert",
			details:    map[string]any{"key": "value", "count": 42},
			routingKey: "test-routing-key",
			severity:   "error",
			dedupKey:   "test-dedup-key",
			statusCode: 202,
			body:       `{"status":"success","message":"Event processed","dedup_key":"test-dedup-key"}`,
		},
		{
			name:       "successful event with minimal fields",
			summary:    "Minimal Alert",
			details:    map[string]any{},
			routingKey: "minimal-key",
			severity:   "warning",
			dedupKey:   "",
			statusCode: 202,
			body:       `{"status":"success"}`,
		},
		{
			name:       "successful event with info severity",
			summary:    "Info Alert",
			details:    map[string]any{"info": "test"},
			routingKey: "info-key",
			severity:   "info",
			dedupKey:   "info-dedup",
			statusCode: 202,
			body:       `{"status":"success","message":"Event processed"}`,
		},
		{
			name:       "successful event with critical severity",
			summary:    "Critical Alert",
			details:    map[string]any{"critical": true},
			routingKey: "critical-key",
			severity:   "critical",
			dedupKey:   "critical-dedup",
			statusCode: 202,
			body:       `{"status":"success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := NewPagerDuty()
			mockClient := createMockHTTPClient(tt.statusCode, tt.body, nil)

			status, err := pd.Send(
				tt.summary,
				tt.details,
				tt.routingKey,
				tt.severity,
				tt.dedupKey,
				mockClient,
			)

			if err != nil {
				t.Errorf("Send() returned unexpected error: %v", err)
			}

			if status == "" {
				t.Error("Send() returned empty status")
			}
		})
	}
}

func TestPagerDuty_Send_WithComplexDetails(t *testing.T) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	details := map[string]any{
		"string_field":  "value",
		"int_field":     123,
		"float_field":   45.67,
		"bool_field":    true,
		"array_field":   []string{"item1", "item2"},
		"nested_object": map[string]any{"nested_key": "nested_value"},
	}

	status, err := pd.Send(
		"Complex Details Test",
		details,
		"test-key",
		"error",
		"complex-dedup",
		mockClient,
	)

	if err != nil {
		t.Errorf("Send() with complex details returned error: %v", err)
	}

	if status == "" {
		t.Error("Send() returned empty status")
	}
}

func TestPagerDuty_Send_WithNilDetails(t *testing.T) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	status, err := pd.Send(
		"Nil Details Test",
		nil,
		"test-key",
		"warning",
		"nil-dedup",
		mockClient,
	)

	if err != nil {
		t.Errorf("Send() with nil details returned error: %v", err)
	}

	if status == "" {
		t.Error("Send() returned empty status")
	}
}

func TestPagerDuty_Send_WithEmptyStrings(t *testing.T) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	status, err := pd.Send(
		"",
		map[string]any{},
		"test-key",
		"",
		"",
		mockClient,
	)

	if err != nil {
		t.Errorf("Send() with empty strings returned error: %v", err)
	}

	if status == "" {
		t.Error("Send() returned empty status")
	}
}

func TestPagerDuty_Send_DifferentSeverities(t *testing.T) {
	severities := []string{"critical", "error", "warning", "info"}
	pd := NewPagerDuty()

	for _, severity := range severities {
		t.Run("severity_"+severity, func(t *testing.T) {
			mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

			status, err := pd.Send(
				"Severity Test",
				map[string]any{"severity_test": severity},
				"test-key",
				severity,
				"severity-dedup-"+severity,
				mockClient,
			)

			if err != nil {
				t.Errorf("Send() with severity %s returned error: %v", severity, err)
			}

			if status == "" {
				t.Errorf("Send() with severity %s returned empty status", severity)
			}
		})
	}
}

// Benchmark tests
func BenchmarkPagerDuty_Send_Simple(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pd.Send(
			"Benchmark Test",
			map[string]any{"iteration": i},
			"bench-key",
			"error",
			"bench-dedup",
			mockClient,
		)
	}
}

func BenchmarkPagerDuty_Send_ComplexDetails(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	details := map[string]any{
		"string_field":  "value",
		"int_field":     123,
		"float_field":   45.67,
		"bool_field":    true,
		"array_field":   []string{"item1", "item2", "item3"},
		"nested_object": map[string]any{"nested_key": "nested_value"},
		"large_array":   make([]int, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pd.Send(
			"Benchmark Complex Test",
			details,
			"bench-key",
			"error",
			"bench-dedup-complex",
			mockClient,
		)
	}
}

func BenchmarkPagerDuty_Send_MinimalData(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pd.Send(
			"Minimal",
			nil,
			"key",
			"error",
			"",
			mockClient,
		)
	}
}

func BenchmarkPagerDuty_Send_LargeSummary(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	// Create a large summary string
	summary := ""
	for i := 0; i < 100; i++ {
		summary += "This is a test alert message. "
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pd.Send(
			summary,
			map[string]any{"test": "data"},
			"bench-key",
			"error",
			"bench-dedup-large",
			mockClient,
		)
	}
}

func BenchmarkPagerDuty_Send_ManyDetails(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	// Create a map with many details
	details := make(map[string]any)
	for i := 0; i < 50; i++ {
		details["field_"+string(rune(i))] = "value_" + string(rune(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pd.Send(
			"Many Details Test",
			details,
			"bench-key",
			"error",
			"bench-dedup-many",
			mockClient,
		)
	}
}

func BenchmarkNewPagerDuty(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewPagerDuty()
	}
}

// Parallel benchmarks
func BenchmarkPagerDuty_Send_Parallel(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = pd.Send(
				"Parallel Benchmark Test",
				map[string]any{"iteration": i},
				"bench-key",
				"error",
				"bench-dedup-parallel",
				mockClient,
			)
			i++
		}
	})
}

func BenchmarkPagerDuty_Send_ParallelComplex(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	details := map[string]any{
		"string_field":  "value",
		"int_field":     123,
		"float_field":   45.67,
		"bool_field":    true,
		"array_field":   []string{"item1", "item2", "item3"},
		"nested_object": map[string]any{"nested_key": "nested_value"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = pd.Send(
				"Parallel Complex Benchmark",
				details,
				"bench-key",
				"error",
				"bench-dedup-parallel-complex",
				mockClient,
			)
		}
	})
}
