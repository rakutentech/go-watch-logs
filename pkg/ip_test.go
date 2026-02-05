// nolint:dupl,gosec,goconst
package pkg

import (
	"fmt"
	"strings"
	"testing"
)

const testCSVData = `16777216,16777471,AU
16777472,16777727,CN
16778240,16779263,CN
16781312,16785407,JP
16785408,16793599,CN
134744064,134750207,AU
134750208,134758399,US
134758400,134766591,CN`

// TestSearchIPAddresses tests the IP address extraction from text
func TestSearchIPAddresses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single IP",
			input:    "Error from 192.168.1.1",
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "multiple IPs",
			input:    "Connection from 192.168.1.1 to 10.0.0.1 failed",
			expected: []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:     "no IPs",
			input:    "This is a log line with no IP addresses",
			expected: []string{},
		},
		{
			name:     "IPs in different formats",
			input:    "Hosts: 8.8.8.8, 255.255.255.0, 0.0.0.0",
			expected: []string{"8.8.8.8", "255.255.255.0", "0.0.0.0"},
		},
		{
			name:     "invalid IP patterns",
			input:    "Not IPs: 999.999.999.999, 1.2.3",
			expected: []string{"999.999.999.999"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "IPs with surrounding text",
			input:    "IP=[192.168.1.100] connected to [10.0.0.5]",
			expected: []string{"192.168.1.100", "10.0.0.5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchIPAddresses(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SearchIPAddresses() returned %d IPs, expected %d", len(result), len(tt.expected))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("SearchIPAddresses() IP[%d] = %s, expected %s", i, ip, tt.expected[i])
				}
			}
		})
	}
}

// TestParseGeoIPCSV tests parsing of GeoIP CSV data
func TestParseGeoIPCSV(t *testing.T) {
	tests := []struct {
		name        string
		csvData     string
		expectError bool
		entryCount  int
	}{
		{
			name:        "valid CSV data",
			csvData:     testCSVData,
			expectError: false,
			entryCount:  8,
		},
		{
			name:        "single entry",
			csvData:     `16777216,16777471,AU`,
			expectError: false,
			entryCount:  1,
		},
		{
			name:        "empty CSV",
			csvData:     "",
			expectError: false,
			entryCount:  0,
		},
		{
			name: "rows with invalid start IP skipped",
			csvData: `invalid,16777471,AU
16777472,16777727,CN`,
			expectError: false,
			entryCount:  1,
		},
		{
			name: "rows with invalid end IP skipped",
			csvData: `16777216,invalid,AU
16777472,16777727,CN`,
			expectError: false,
			entryCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := ParseGeoIPCSV(tt.csvData)
			if tt.expectError && err == nil {
				t.Error("ParseGeoIPCSV() expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("ParseGeoIPCSV() unexpected error: %v", err)
				return
			}
			if !tt.expectError && db != nil && len(db.entries) != tt.entryCount {
				t.Errorf("ParseGeoIPCSV() got %d entries, expected %d", len(db.entries), tt.entryCount)
			}
		})
	}
}

// TestLookupIP tests IP address lookups
func TestLookupIP(t *testing.T) {
	db, err := ParseGeoIPCSV(testCSVData)
	if err != nil {
		t.Fatalf("Failed to parse test CSV: %v", err)
	}

	tests := []struct {
		name            string
		ip              string
		expectedCode    string
		expectedCountry string
		expectError     bool
	}{
		{
			name:            "valid IP in Australia range (first)",
			ip:              "1.0.0.1",
			expectedCode:    "AU",
			expectedCountry: "AU",
			expectError:     false,
		},
		{
			name:            "valid IP in China range",
			ip:              "1.0.1.1",
			expectedCode:    "CN",
			expectedCountry: "CN",
			expectError:     false,
		},
		{
			name:            "valid IP in Japan range",
			ip:              "1.0.20.1",
			expectedCode:    "JP",
			expectedCountry: "JP",
			expectError:     false,
		},
		{
			name:            "valid IP in US range",
			ip:              "8.8.50.1",
			expectedCode:    "US",
			expectedCountry: "US",
			expectError:     false,
		},
		{
			name:            "valid IP in Australia range (second)",
			ip:              "8.8.8.8",
			expectedCode:    "AU",
			expectedCountry: "AU",
			expectError:     false,
		},
		{
			name:            "IP not in database",
			ip:              "200.200.200.200",
			expectedCode:    "ZZ",
			expectedCountry: "Unknown",
			expectError:     false,
		},
		{
			name:            "invalid IP format",
			ip:              "invalid",
			expectedCode:    "",
			expectedCountry: "",
			expectError:     true,
		},
		{
			name:            "empty IP",
			ip:              "",
			expectedCode:    "",
			expectedCountry: "",
			expectError:     true,
		},
		{
			name:            "IPv6 address",
			ip:              "2001:4860:4860::8888",
			expectedCode:    "",
			expectedCountry: "",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, country, err := db.LookupIP(tt.ip)
			if tt.expectError && err == nil {
				t.Error("LookupIP() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("LookupIP() unexpected error: %v", err)
			}
			if !tt.expectError {
				if code != tt.expectedCode {
					t.Errorf("LookupIP() code = %s, expected %s", code, tt.expectedCode)
				}
				if country != tt.expectedCountry {
					t.Errorf("LookupIP() country = %s, expected %s", country, tt.expectedCountry)
				}
			}
		})
	}
}

// TestIPToUint32 tests IP to uint32 conversion
func TestIPToUint32(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{
			name:     "zero IP",
			ip:       "0.0.0.0",
			expected: 0,
		},
		{
			name:     "localhost",
			ip:       "127.0.0.1",
			expected: 2130706433,
		},
		{
			name:     "private IP",
			ip:       "192.168.1.1",
			expected: 3232235777,
		},
		{
			name:     "Google DNS",
			ip:       "8.8.8.8",
			expected: 134744072,
		},
		{
			name:     "max IP",
			ip:       "255.255.255.255",
			expected: 4294967295,
		},
		{
			name:     "invalid IP",
			ip:       "invalid",
			expected: 0,
		},
		{
			name:     "empty string",
			ip:       "",
			expected: 0,
		},
		{
			name:     "IPv6 address",
			ip:       "2001:4860:4860::8888",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToUint32(tt.ip)
			if result != tt.expected {
				t.Errorf("ipToUint32(%s) = %d, expected %d", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestLookupMultipleIPs tests looking up multiple IPs
func TestLookupMultipleIPs(t *testing.T) {
	db, err := ParseGeoIPCSV(testCSVData)
	if err != nil {
		t.Fatalf("Failed to parse test CSV: %v", err)
	}

	tests := []struct {
		name          string
		ips           []string
		expectedCount int
		checkResults  bool
		expectedCodes []string
	}{
		{
			name:          "single valid IP",
			ips:           []string{"1.0.0.1"},
			expectedCount: 1,
			checkResults:  true,
			expectedCodes: []string{"AU"},
		},
		{
			name:          "multiple valid IPs",
			ips:           []string{"1.0.0.1", "1.0.1.1", "8.8.50.1"},
			expectedCount: 3,
			checkResults:  true,
			expectedCodes: []string{"AU", "CN", "US"},
		},
		{
			name:          "empty slice",
			ips:           []string{},
			expectedCount: 0,
			checkResults:  false,
		},
		{
			name:          "mixed valid and invalid",
			ips:           []string{"1.0.0.1", "invalid", "8.8.50.1"},
			expectedCount: 3,
			checkResults:  true,
			expectedCodes: []string{"AU", "ZZ", "US"},
		},
		{
			name:          "all invalid",
			ips:           []string{"invalid", "not-an-ip"},
			expectedCount: 2,
			checkResults:  true,
			expectedCodes: []string{"ZZ", "ZZ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := db.LookupMultipleIPs(tt.ips)
			if len(results.Results) != tt.expectedCount {
				t.Errorf("LookupMultipleIPs() returned %d results, expected %d", len(results.Results), tt.expectedCount)
			}
			if tt.checkResults {
				for i, result := range results.Results {
					if result.IP != tt.ips[i] {
						t.Errorf("Result[%d] IP = %s, expected %s", i, result.IP, tt.ips[i])
					}
					if result.CountryCode != tt.expectedCodes[i] {
						t.Errorf("Result[%d] CountryCode = %s, expected %s", i, result.CountryCode, tt.expectedCodes[i])
					}
				}
			}
		})
	}
}

// TestGetCountryCounts tests country aggregation
func TestGetCountryCounts(t *testing.T) {
	db, err := ParseGeoIPCSV(testCSVData)
	if err != nil {
		t.Fatalf("Failed to parse test CSV: %v", err)
	}

	tests := []struct {
		name           string
		ips            []string
		expectedCounts map[string]int
	}{
		{
			name: "single country",
			ips:  []string{"1.0.0.1", "1.0.0.2"},
			expectedCounts: map[string]int{
				"AU": 2,
			},
		},
		{
			name: "multiple countries",
			ips:  []string{"1.0.0.1", "1.0.1.1", "8.8.50.1"},
			expectedCounts: map[string]int{
				"AU": 1,
				"CN": 1,
				"US": 1,
			},
		},
		{
			name: "repeated countries",
			ips:  []string{"1.0.0.1", "8.8.8.8", "8.8.50.1", "8.8.51.1"},
			expectedCounts: map[string]int{
				"AU": 2,
				"US": 2,
			},
		},
		{
			name:           "empty slice",
			ips:            []string{},
			expectedCounts: map[string]int{},
		},
		{
			name: "with invalid IPs",
			ips:  []string{"1.0.0.1", "invalid", "not-an-ip"},
			expectedCounts: map[string]int{
				"AU":      1,
				"Unknown": 2,
			},
		},
		{
			name: "all unknown",
			ips:  []string{"200.200.200.200", "201.201.201.201"},
			expectedCounts: map[string]int{
				"Unknown": 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counts := db.GetCountryCounts(tt.ips)
			if len(counts) != len(tt.expectedCounts) {
				t.Errorf("GetCountryCounts() returned %d countries, expected %d", len(counts), len(tt.expectedCounts))
			}
			for country, expectedCount := range tt.expectedCounts {
				if count, ok := counts[country]; !ok {
					t.Errorf("GetCountryCounts() missing country %s", country)
				} else if count != expectedCount {
					t.Errorf("GetCountryCounts() country %s count = %d, expected %d", country, count, expectedCount)
				}
			}
		})
	}
}

// TestGeoIPDatabase_EdgeCases tests edge cases in GeoIP lookups
func TestGeoIPDatabase_EdgeCases(t *testing.T) {
	db, err := ParseGeoIPCSV(testCSVData)
	if err != nil {
		t.Fatalf("Failed to parse test CSV: %v", err)
	}

	t.Run("boundary IP - start of range", func(t *testing.T) {
		// 16777216 = 1.0.0.0 (start of first range)
		code, country, err := db.LookupIP("1.0.0.0")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if code != "AU" || country != "AU" {
			t.Errorf("Expected AU/AU, got %s/%s", code, country)
		}
	})

	t.Run("boundary IP - end of range", func(t *testing.T) {
		// 16777471 = 1.0.0.255 (end of first range)
		code, country, err := db.LookupIP("1.0.0.255")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if code != "AU" || country != "AU" {
			t.Errorf("Expected AU/AU, got %s/%s", code, country)
		}
	})

	t.Run("IP just outside range", func(t *testing.T) {
		// IP between ranges should return Unknown
		code, country, err := db.LookupIP("1.0.3.0")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if code != "ZZ" || country != "Unknown" {
			t.Errorf("Expected ZZ/Unknown, got %s/%s", code, country)
		}
	})
}

// TestIPLookupResult tests IPLookupResult structure
func TestIPLookupResult(t *testing.T) {
	db, err := ParseGeoIPCSV(testCSVData)
	if err != nil {
		t.Fatalf("Failed to parse test CSV: %v", err)
	}

	t.Run("result fields for valid IP", func(t *testing.T) {
		results := db.LookupMultipleIPs([]string{"1.0.0.1"})
		if len(results.Results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results.Results))
		}
		result := results.Results[0]
		if result.IP != "1.0.0.1" {
			t.Errorf("Expected IP 1.0.0.1, got %s", result.IP)
		}
		if result.CountryCode != "AU" {
			t.Errorf("Expected code AU, got %s", result.CountryCode)
		}
		if result.CountryName != "AU" {
			t.Errorf("Expected country AU, got %s", result.CountryName)
		}
		if result.Error != nil {
			t.Errorf("Expected no error, got %v", result.Error)
		}
	})

	t.Run("result fields for invalid IP", func(t *testing.T) {
		results := db.LookupMultipleIPs([]string{"invalid"})
		if len(results.Results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results.Results))
		}
		result := results.Results[0]
		if result.IP != "invalid" {
			t.Errorf("Expected IP invalid, got %s", result.IP)
		}
		if result.CountryCode != "ZZ" {
			t.Errorf("Expected code ZZ, got %s", result.CountryCode)
		}
		if result.CountryName != "Unknown" {
			t.Errorf("Expected country Unknown, got %s", result.CountryName)
		}
		if result.Error == nil {
			t.Error("Expected error for invalid IP")
		}
	})
}

// TestParseGeoIPCSV_DataIntegrity tests that parsed data maintains integrity
func TestParseGeoIPCSV_DataIntegrity(t *testing.T) {
	csvData := `16777216,16777471,US`
	db, err := ParseGeoIPCSV(csvData)
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(db.entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(db.entries))
	}

	entry := db.entries[0]
	if entry.StartIP != 16777216 {
		t.Errorf("Expected StartIP 16777216, got %d", entry.StartIP)
	}
	if entry.EndIP != 16777471 {
		t.Errorf("Expected EndIP 16777471, got %d", entry.EndIP)
	}
	if entry.CountryCode != "US" {
		t.Errorf("Expected CountryCode US, got %s", entry.CountryCode)
	}
	if entry.CountryName != "US" {
		t.Errorf("Expected CountryName 'US', got %s", entry.CountryName)
	}
}

// TestSearchIPAddresses_RealWorldLogs tests with realistic log formats
func TestSearchIPAddresses_RealWorldLogs(t *testing.T) {
	tests := []struct {
		name     string
		logLine  string
		expected []string
	}{
		{
			name:     "Apache access log",
			logLine:  `192.168.1.1 - - [06/Jan/2025:10:15:23 +0000] "GET /index.html HTTP/1.1" 200 1234`,
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "Nginx error log",
			logLine:  `2025/01/06 10:15:23 [error] 12345#0: *1 connect() failed (111: Connection refused) while connecting to upstream, client: 10.0.0.5`,
			expected: []string{"10.0.0.5"},
		},
		{
			name:     "Firewall log with multiple IPs",
			logLine:  `2025-01-06T10:15:23Z DENY TCP 192.168.1.100:54321 -> 8.8.8.8:443`,
			expected: []string{"192.168.1.100", "8.8.8.8"},
		},
		{
			name:     "Application log with JSON",
			logLine:  `{"timestamp":"2025-01-06T10:15:23Z","level":"error","msg":"connection failed","src_ip":"172.16.0.1","dst_ip":"172.16.0.254"}`,
			expected: []string{"172.16.0.1", "172.16.0.254"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchIPAddresses(tt.logLine)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d IPs, got %d", len(tt.expected), len(result))
				return
			}
			for i, ip := range result {
				if ip != tt.expected[i] {
					t.Errorf("IP[%d] = %s, expected %s", i, ip, tt.expected[i])
				}
			}
		})
	}
}

// TestParseGeoIPCSV_LargeDataset tests with a larger dataset
func TestParseGeoIPCSV_LargeDataset(t *testing.T) {
	// Generate a larger CSV dataset
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		start := uint32(16777216 + i*256)
		end := start + 255
		builder.WriteString(fmt.Sprintf("%d,%d,US\n", start, end))
	}

	db, err := ParseGeoIPCSV(builder.String())
	if err != nil {
		t.Fatalf("Failed to parse large CSV: %v", err)
	}

	if len(db.entries) != 1000 {
		t.Errorf("Expected 1000 entries, got %d", len(db.entries))
	}

	// Test lookup in the middle of the dataset
	code, country, err := db.LookupIP("1.0.128.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if code != "US" || country != "US" {
		t.Errorf("Expected US/US, got %s/%s", code, country)
	}
}

func setupTestDB(b *testing.B) *GeoIPDatabase {
	db, err := ParseGeoIPCSV(testCSVData)
	if err != nil {
		b.Fatalf("Failed to parse test CSV: %v", err)
	}
	return db
}

func BenchmarkSearchIPAddresses(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "single_ip",
			input: "Error from 192.168.1.1",
		},
		{
			name:  "multiple_ips",
			input: "Connection from 192.168.1.1 to 10.0.0.1 failed, redirecting to 172.16.0.1",
		},
		{
			name:  "no_ips",
			input: "This is a log line with no IP addresses",
		},
		{
			name:  "large_text",
			input: "Lorem ipsum dolor sit amet 192.168.1.1 consectetur adipiscing elit 10.0.0.1 sed do eiusmod tempor incididunt ut labore et dolore magna aliqua 172.16.0.1 ut enim ad minim veniam quis nostrud exercitation ullamco laboris 8.8.8.8 nisi ut aliquip ex ea commodo consequat",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				SearchIPAddresses(tc.input)
			}
		})
	}
}

func BenchmarkParseGeoIPCSV(b *testing.B) {
	testCases := []struct {
		name string
		data string
	}{
		{
			name: "small_dataset",
			data: testCSVData,
		},
		{
			name: "single_entry",
			data: `16777216,16777471,16777216,16777471,AU,Australia,Australia`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := ParseGeoIPCSV(tc.data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkLookupIP(b *testing.B) {
	db := setupTestDB(b)

	testCases := []struct {
		name string
		ip   string
	}{
		{
			name: "first_range",
			ip:   "1.0.0.1",
		},
		{
			name: "middle_range",
			ip:   "1.0.36.1",
		},
		{
			name: "last_range",
			ip:   "8.8.8.8",
		},
		{
			name: "not_found",
			ip:   "255.255.255.255",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _, _ = db.LookupIP(tc.ip)
			}
		})
	}
}

func BenchmarkLookupMultipleIPs(b *testing.B) {
	db := setupTestDB(b)

	testCases := []struct {
		name string
		ips  []string
	}{
		{
			name: "single_ip",
			ips:  []string{"1.0.0.1"},
		},
		{
			name: "five_ips",
			ips:  []string{"1.0.0.1", "1.0.1.1", "1.0.36.1", "8.8.8.8", "8.8.4.4"},
		},
		{
			name: "ten_ips",
			ips:  []string{"1.0.0.1", "1.0.1.1", "1.0.36.1", "1.1.1.1", "8.8.8.8", "8.8.4.4", "1.2.3.4", "5.6.7.8", "9.10.11.12", "13.14.15.16"},
		},
		{
			name: "mixed_valid_invalid",
			ips:  []string{"1.0.0.1", "invalid", "8.8.8.8", "999.999.999.999", "1.0.36.1"},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				db.LookupMultipleIPs(tc.ips)
			}
		})
	}
}

func BenchmarkGetCountryCounts(b *testing.B) {
	db := setupTestDB(b)

	testCases := []struct {
		name string
		ips  []string
	}{
		{
			name: "single_ip",
			ips:  []string{"1.0.0.1"},
		},
		{
			name: "same_country",
			ips:  []string{"1.0.0.1", "1.0.0.2", "1.0.0.3"},
		},
		{
			name: "different_countries",
			ips:  []string{"1.0.0.1", "1.0.1.1", "1.0.36.1", "8.8.8.8"},
		},
		{
			name: "ten_ips_mixed",
			ips:  []string{"1.0.0.1", "1.0.0.2", "1.0.1.1", "1.0.36.1", "1.0.36.2", "8.8.8.8", "8.8.8.9", "8.8.8.10", "8.8.4.4", "8.8.4.5"},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				db.GetCountryCounts(tc.ips)
			}
		})
	}
}

func BenchmarkIPToUint32(b *testing.B) {
	testCases := []struct {
		name string
		ip   string
	}{
		{
			name: "low_address",
			ip:   "1.0.0.1",
		},
		{
			name: "mid_address",
			ip:   "128.128.128.128",
		},
		{
			name: "high_address",
			ip:   "255.255.255.254",
		},
		{
			name: "invalid_ip",
			ip:   "invalid",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ipToUint32(tc.ip)
			}
		})
	}
}

// BenchmarkEndToEndGeoLookup simulates a realistic end-to-end workflow
func BenchmarkEndToEndGeoLookup(b *testing.B) {
	db := setupTestDB(b)

	testCases := []struct {
		name    string
		logLine string
	}{
		{
			name:    "single_ip_log",
			logLine: "2025-01-06 10:15:23 ERROR Connection failed from 1.0.0.1",
		},
		{
			name:    "multiple_ips_log",
			logLine: "2025-01-06 10:15:23 INFO Request from 1.0.0.1 proxied through 8.8.8.8 to 1.0.36.1",
		},
		{
			name:    "no_ips_log",
			logLine: "2025-01-06 10:15:23 WARNING Memory usage exceeded threshold",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Extract IPs
				ips := SearchIPAddresses(tc.logLine)
				// Lookup countries
				_ = db.LookupMultipleIPs(ips)
			}
		})
	}
}

// BenchmarkConcurrentLookup tests performance under concurrent access
func BenchmarkConcurrentLookup(b *testing.B) {
	db := setupTestDB(b)
	ips := []string{"1.0.0.1", "1.0.1.1", "1.0.36.1", "8.8.8.8", "8.8.4.4"}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ip := ips[i%len(ips)]
			_, _, _ = db.LookupIP(ip)
			i++
		}
	})
}
