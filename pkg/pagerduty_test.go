package pkg

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

const (
	severityError    = "error"
	severityWarning  = "warning"
	severityInfo     = "info"
	severityCritical = "critical"

	fieldString    = "string_field"
	fieldInt       = "int_field"
	fieldFloat     = "float_field"
	fieldBool      = "bool_field"
	fieldArray     = "array_field"
	fieldNested    = "nested_key"
	fieldNestedObj = "nested_object"

	testValue     = "value"
	testItem1     = "item1"
	testItem2     = "item2"
	testNestedVal = "nested_value"
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
			details:    map[string]any{"key": testValue, "count": 42},
			routingKey: "test-routing-key",
			severity:   severityError,
			dedupKey:   "test-dedup-key",
			statusCode: 202,
			body:       `{"status":"success","message":"Event processed","dedup_key":"test-dedup-key"}`,
		},
		{
			name:       "successful event with minimal fields",
			summary:    "Minimal Alert",
			details:    map[string]any{},
			routingKey: "minimal-key",
			severity:   severityWarning,
			dedupKey:   "",
			statusCode: 202,
			body:       `{"status":"success"}`,
		},
		{
			name:       "successful event with info severity",
			summary:    "Info Alert",
			details:    map[string]any{severityInfo: "test"},
			routingKey: "info-key",
			severity:   severityInfo,
			dedupKey:   "info-dedup",
			statusCode: 202,
			body:       `{"status":"success","message":"Event processed"}`,
		},
		{
			name:       "successful event with critical severity",
			summary:    "Critical Alert",
			details:    map[string]any{severityCritical: true},
			routingKey: "critical-key",
			severity:   severityCritical,
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
		fieldString:  testValue,
		fieldInt:     123,
		fieldFloat:   45.67,
		fieldBool:    true,
		fieldArray:   []string{testItem1, testItem2},
		fieldNestedObj: map[string]any{fieldNested: testNestedVal},
	}

	status, err := pd.Send(
		"Complex Details Test",
		details,
		"test-key",
		severityError,
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
		severityWarning,
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
	severities := []string{severityCritical, severityError, severityWarning, severityInfo}
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
			severityError,
			"bench-dedup",
			mockClient,
		)
	}
}

func BenchmarkPagerDuty_Send_ComplexDetails(b *testing.B) {
	pd := NewPagerDuty()
	mockClient := createMockHTTPClient(202, `{"status":"success"}`, nil)

	details := map[string]any{
		fieldString:  testValue,
		fieldInt:     123,
		fieldFloat:   45.67,
		fieldBool:    true,
		fieldArray:   []string{testItem1, testItem2, "item3"},
		fieldNestedObj: map[string]any{fieldNested: testNestedVal},
		"large_array":   make([]int, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pd.Send(
			"Benchmark Complex Test",
			details,
			"bench-key",
			severityError,
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
			severityError,
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
			severityError,
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
			severityError,
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
				severityError,
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
		fieldString:  testValue,
		fieldInt:     123,
		fieldFloat:   45.67,
		fieldBool:    true,
		fieldArray:   []string{testItem1, testItem2, "item3"},
		fieldNestedObj: map[string]any{fieldNested: testNestedVal},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = pd.Send(
				"Parallel Complex Benchmark",
				details,
				"bench-key",
				severityError,
				"bench-dedup-parallel-complex",
				mockClient,
			)
		}
	})
}
