package pkg

import (
	"bufio"
	"io"
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
	maxBufferMB     int
	lastLineNum     int
	lastFileSize    int64
	timestampNow    string
	streak          int
}

const limitCountryCount = 25

const previewLineMaxLength = 500

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

	regMatch, err := regexp.Compile(w.matchPattern)
	if err != nil {
		return nil, err
	}
	regIgnore, err := regexp.Compile(w.ignorePattern)
	if err != nil {
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

		if w.ignorePattern != "" && regIgnore.Match(line) {
			continue
		}
		if regMatch.Match(line) {
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
