package pkg

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/patrickmn/go-cache"
)

type Watcher struct {
	cache           *cache.Cache
	filePath        string
	geoIPDB         *GeoIPDatabase
	lastLineKey     string
	lastFileSizeKey string
	errorHistoryKey string
	scanCountKey    string
	matchPattern    string
	ignorePattern   string
	regexMatch      []*regexp.Regexp // Pre-compiled match regexes
	regexIgnore     []*regexp.Regexp // Pre-compiled ignore regexes
	maxBufferMB     int
	lastLineNum     int
	lastFileSize    int64
	timestampNow    string
	streak          int
}

const limitCountryCount = 25

const previewLineMaxLength = 500

// Pattern splitting thresholds
const (
	patternSplitThreshold = 500 // Minimum length to consider splitting
	patternChunkSize      = 300 // Target size for each chunk
)

// splitAndCompilePattern splits a long pattern by | and compiles into multiple regexes
func splitAndCompilePattern(pattern string) ([]*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}

	// If pattern is short, compile as-is
	if len(pattern) < patternSplitThreshold {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return []*regexp.Regexp{re}, nil
	}

	// Split long patterns by | for efficiency
	parts := splitPattern(pattern)

	// Calculate how many parts per regex based on total length
	totalLen := len(pattern)
	numChunks := (totalLen + patternChunkSize - 1) / patternChunkSize
	if numChunks < 1 {
		numChunks = 1
	}
	if numChunks > len(parts) {
		numChunks = len(parts)
	}

	partsPerChunk := (len(parts) + numChunks - 1) / numChunks
	if partsPerChunk < 1 {
		partsPerChunk = 1
	}

	var regexes []*regexp.Regexp
	for i := 0; i < len(parts); i += partsPerChunk {
		end := i + partsPerChunk
		if end > len(parts) {
			end = len(parts)
		}

		// Join the parts back together
		chunkPattern := ""
		for j := i; j < end; j++ {
			if j > i {
				chunkPattern += "|"
			}
			chunkPattern += parts[j]
		}

		re, err := regexp.Compile(chunkPattern)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, re)
	}

	if len(regexes) > 1 {
		slog.Warn("Regex was long, splitting into parts", "originalLength", len(pattern), "regexCount", len(regexes))
		slog.Info("Compiled regex patterns", "patterns", fmt.Sprintf("%q", regexes))
	}

	return regexes, nil
}

// splitPattern splits a regex pattern by | separators
// Escaped pipes (\|) are NOT used as split points - they are kept as part of the pattern
func splitPattern(pattern string) []string {
	if pattern == "" {
		return []string{}
	}

	var parts []string
	var current string
	escaped := false

	for _, ch := range pattern {
		if escaped {
			current += string(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			current += string(ch)
			escaped = true
			continue
		}

		if ch == '|' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			continue
		}

		current += string(ch)
	}

	if current != "" {
		parts = append(parts, current)
	}

	// If no splits found, return the original pattern
	if len(parts) == 0 {
		return []string{pattern}
	}

	return parts
}

func NewWatcher(
	filePath string,
	f Flags,
	c *cache.Cache,
	geoIPDB *GeoIPDatabase,
) (*Watcher, error) {
	now := time.Now()

	watcher := &Watcher{
		cache:           c,
		filePath:        filePath,
		geoIPDB:         geoIPDB,
		matchPattern:    f.Match,
		ignorePattern:   f.Ignore,
		lastLineKey:     "lk-" + filePath,
		lastFileSizeKey: "sk-" + filePath,
		errorHistoryKey: "eh-" + filePath,
		scanCountKey:    "sc-" + filePath,
		timestampNow:    now.Format("2006-01-02 15:04:05"),
		maxBufferMB:     f.MaxBufferMB,
		streak:          DisplayableStreakNumber(f.Streak),
	}

	// Pre-compile match regexes
	var err error
	watcher.regexMatch, err = splitAndCompilePattern(f.Match)
	if err != nil {
		return nil, err
	}

	// Pre-compile ignore regexes
	watcher.regexIgnore, err = splitAndCompilePattern(f.Ignore)
	if err != nil {
		return nil, err
	}

	if err := watcher.loadState(); err != nil {
		return nil, err
	}

	return watcher, nil
}

type ScanResult struct {
	FilePath      string
	FileInfo      os.FileInfo
	ErrorCount    int
	ErrorPercent  float64
	Severity   string
	LinesRead     int
	FirstLine     string
	FirstDate     string
	CountryCounts map[string]int
	PreviewLine   string
	LastLine      string
	LastDate      string
	Streak        []int // History of error counts for this file path
	ScanCount     int   // Total number of scans performed
}

func (r *ScanResult) IsFirstScan() bool {
	return r.ScanCount == 1
}

func (w *Watcher) Scan() (*ScanResult, error) {
	matchCounts := 0
	firstLine := ""
	lastLine := ""
	previewLine := ""

	fileInfo, err := os.Stat(w.filePath)
	if err != nil {
		return nil, err
	}

	currentFileSize := fileInfo.Size()

	// Detect log rotation
	if currentFileSize+2 < w.lastFileSize {
		w.lastFileSize = 0
	}

	file, err := os.Open(w.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.Seek(w.lastFileSize, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	if w.maxBufferMB > 0 {
		// For large lines
		scanner.Buffer(make([]byte, 0, 64*1024), w.maxBufferMB*1024*1024)
	}
	currentLineNum := 1
	linesRead := 0
	bytesRead := w.lastFileSize
	isFirstScan := w.getScanCount() == 0
	countryCounts := make(map[string]int)

	for scanner.Scan() {
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // Adding 1 for the newline character
		currentLineNum++
		linesRead = currentLineNum - w.lastLineNum
		// Convert to positive number
		if linesRead < 0 {
			linesRead = -linesRead
		}

		if isFirstScan {
			continue
		}

		if w.matchesAny(w.regexIgnore, line) {
			continue
		}
		if w.matchesAny(w.regexMatch, line) {
			lineStr := string(line)

			if len(countryCounts) < limitCountryCount {
				cc := w.geoIPDB.GetCountryCounts(SearchIPAddresses(lineStr))
				for country, count := range cc {
					countryCounts[country] += count
				}
			}

			if firstLine == "" {
				firstLine = lineStr
			}
			if len(previewLine) < previewLineMaxLength {
				previewLine += lineStr + "\n\r"
			}
			lastLine = lineStr
			matchCounts++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	matchPercentage := 0.0
	if linesRead > 0 {
		matchPercentage = float64(matchCounts) * 100 / float64(linesRead)
		if matchPercentage > 100 {
			matchPercentage = 100
		}
	}

	// Restrict to two decimal places
	matchPercentage = float64(int(matchPercentage*100)) / 100
	severity := "error"
	if matchPercentage >= 50 {
		severity = "critical"
	}

	w.lastLineNum = currentLineNum
	w.lastFileSize = bytesRead

	// Update scan count
	w.incrementScanCount()

	// Update error history
	w.updateErrorHistory(matchCounts)

	// Save state
	if err := w.saveState(); err != nil {
		return nil, err
	}

	// Get the error history
	errorHistory := w.getErrorHistory()

	// Get the scan count
	scanCount := w.getScanCount()

	return &ScanResult{
		ErrorCount:    matchCounts,
		FirstDate:     SearchDate(firstLine),
		LastDate:      SearchDate(lastLine),
		FirstLine:     firstLine,
		PreviewLine:   previewLine,
		LastLine:      lastLine,
		FilePath:      w.filePath,
		FileInfo:      fileInfo,
		ErrorPercent:  matchPercentage,
		Severity:      severity,
		LinesRead:     linesRead,
		Streak:        errorHistory,
		ScanCount:     scanCount,
		CountryCounts: countryCounts,
	}, nil
}

// matchesAny checks if the line matches any of the regexes in the slice
func (w *Watcher) matchesAny(regexes []*regexp.Regexp, line []byte) bool {
	for _, re := range regexes {
		if re.Match(line) {
			return true
		}
	}
	return false
}

func (w *Watcher) loadState() error {
	if value, found := w.cache.Get(w.lastLineKey); found {
		w.lastLineNum = value.(int)
	}
	if value, found := w.cache.Get(w.lastFileSizeKey); found {
		w.lastFileSize = value.(int64)
	}
	return nil
}

func (w *Watcher) saveState() error {
	w.cache.Set(w.lastLineKey, w.lastLineNum, cache.DefaultExpiration)
	w.cache.Set(w.lastFileSizeKey, w.lastFileSize, cache.DefaultExpiration)
	return nil
}

func (w *Watcher) updateErrorHistory(newErrorCount int) {
	if w.getScanCount() == 1 {
		return
	}
	var history []int
	if value, found := w.cache.Get(w.errorHistoryKey); found {
		history = value.([]int)
	}

	// Add the new error count and limit the history size
	history = append(history, newErrorCount)
	if len(history) > w.streak {
		history = history[len(history)-w.streak:]
	}

	w.cache.Set(w.errorHistoryKey, history, cache.DefaultExpiration)
}

func (w *Watcher) getErrorHistory() []int {
	if value, found := w.cache.Get(w.errorHistoryKey); found {
		return value.([]int)
	}
	return []int{}
}

func (w *Watcher) incrementScanCount() {
	count := 0
	if value, found := w.cache.Get(w.scanCountKey); found {
		count = value.(int)
	}
	count++
	w.cache.Set(w.scanCountKey, count, cache.DefaultExpiration)
}

func (w *Watcher) getScanCount() int {
	if value, found := w.cache.Get(w.scanCountKey); found {
		return value.(int)
	}
	return 0
}

func (w *Watcher) Close() error {
	return nil
}
