// nolint:goconst
package pkg

import (
	"encoding/csv"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

func SearchIPAddresses(input string) []string {
	var ips []string
	ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	matches := ipRegex.FindAllString(input, -1)
	ips = append(ips, matches...)
	return ips
}

// GeoIPDatabase holds the parsed GeoIP data
type GeoIPDatabase struct {
	entries []GeoIPEntry
}

// GeoIPEntry represents a single GeoIP range entry
type GeoIPEntry struct {
	StartIP     uint32
	EndIP       uint32
	CountryCode string
	CountryName string
}

// ParseGeoIPCSV parses the embedded GeoIP CSV data
func ParseGeoIPCSV(csvData string) (*GeoIPDatabase, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	db := &GeoIPDatabase{
		entries: make([]GeoIPEntry, 0, len(records)),
	}

	for _, record := range records {
		if len(record) < 7 {
			continue
		}

		startIP, err := strconv.ParseUint(record[0], 10, 32)
		if err != nil {
			continue
		}

		endIP, err := strconv.ParseUint(record[1], 10, 32)
		if err != nil {
			continue
		}

		entry := GeoIPEntry{
			StartIP:     uint32(startIP),
			EndIP:       uint32(endIP),
			CountryCode: record[4],
			CountryName: record[6],
		}

		db.entries = append(db.entries, entry)
	}

	return db, nil
}

// LookupIP finds the country for a given IP address
func (db *GeoIPDatabase) LookupIP(ip string) (string, string, error) {
	ipNum := ipToUint32(ip)
	if ipNum == 0 {
		return "", "", fmt.Errorf("invalid IP address: %s", ip)
	}

	// Binary search for the IP range
	left, right := 0, len(db.entries)-1
	for left <= right {
		mid := (left + right) / 2
		entry := db.entries[mid]

		if ipNum >= entry.StartIP && ipNum <= entry.EndIP {
			return entry.CountryCode, entry.CountryName, nil
		}

		if ipNum < entry.StartIP {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	return "ZZ", "Unknown", nil
}

// ipToUint32 converts an IP address string to uint32
func ipToUint32(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}

	ip = ip.To4()
	if ip == nil {
		return 0
	}

	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// IPLookupResult represents the result of looking up an IP address
type IPLookupResult struct {
	IP          string
	CountryCode string
	CountryName string
	Error       error
}

// IPLookupResults represents the results of looking up multiple IP addresses
type IPLookupResults struct {
	Results []IPLookupResult
}

// LookupMultipleIPs looks up countries for multiple IP addresses
func (db *GeoIPDatabase) LookupMultipleIPs(ips []string) IPLookupResults {
	results := IPLookupResults{
		Results: make([]IPLookupResult, 0, len(ips)),
	}

	for _, ip := range ips {
		code, country, err := db.LookupIP(ip)
		result := IPLookupResult{
			IP:          ip,
			CountryCode: code,
			CountryName: country,
			Error:       err,
		}
		if err != nil {
			result.CountryName = "Unknown"
			result.CountryCode = "ZZ"
		}
		results.Results = append(results.Results, result)
	}
	return results
}

// GetCountryCounts aggregates IP lookup results by country name and returns counts
func (db *GeoIPDatabase) GetCountryCounts(ips []string) map[string]int {
	countryCounts := make(map[string]int)

	lookupResults := db.LookupMultipleIPs(ips)
	for _, result := range lookupResults.Results {
		countryCounts[result.CountryName]++
	}

	return countryCounts
}
