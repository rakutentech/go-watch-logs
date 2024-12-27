package pkg

import (
	"bufio"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"
)

type Watcher struct {
	db              *sql.DB
	dbName          string // full path
	filePath        string
	anomalyKey      string
	lastLineKey     string
	lastFileSizeKey string
	matchPattern    string
	ignorePattern   string
	lastLineNum     int
	lastFileSize    int64
	anomalizer      *Anomalizer
	anomaly         bool
	anomalyWindow   int
}

func NewWatcher(
	filePath string,
	f Flags,
) (*Watcher, error) {
	dbName := f.DBPath + ".sqlite"
	db, err := InitDB(dbName)
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		db:              db,
		dbName:          dbName,
		filePath:        filePath,
		anomaly:         f.Anomaly,
		anomalizer:      NewAnomalizer(),
		anomalyWindow:   f.AnomalyWindowDays,
		matchPattern:    f.Match,
		ignorePattern:   f.Ignore,
		anomalyKey:      "anm-" + filePath,
		lastLineKey:     "llk-" + filePath,
		lastFileSizeKey: "llks-" + filePath,
	}
	if err := watcher.loadState(); err != nil {
		return nil, err
	}

	return watcher, nil
}

type ScanResult struct {
	FilePath     string
	FileInfo     os.FileInfo
	ErrorCount   int
	ErrorPercent float64
	LinesRead    int
	FirstLine    string
	FirstDate    string
	PreviewLine  string
	LastLine     string
	LastDate     string
}

var lines = []string{}

func (w *Watcher) Scan() (*ScanResult, error) {
	errorCounts := 0
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
	currentLineNum := 1
	linesRead := 0
	bytesRead := w.lastFileSize

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // Adding 1 for the newline character
		currentLineNum++
		linesRead = currentLineNum - w.lastLineNum
		// convert to positive number
		if linesRead < 0 {
			linesRead = -linesRead
		}
		// slog.Debug("Scanning line", "line", string(line), "lineNum", currentLineNum, "linesRead", linesRead)
		if w.ignorePattern != "" && regIgnore.Match(line) {
			continue
		}

		// anomaly insertion
		if w.anomaly {
			match := regMatch.FindAllString(string(line), -1)
			var exactMatch string
			if len(match) >= 1 {
				exactMatch = match[0]
			}
			if exactMatch != "" {
				slog.Info("Match found", "line", string(line), "match", exactMatch)
				w.anomalizer.MemSafeCount(exactMatch)
			}
			continue // no need to go for match as this is anomaly check only
		}

		if regMatch.Match(line) {
			lineStr := string(line)
			if firstLine == "" {
				firstLine = lineStr
			}
			if len(previewLine) < 1000 {
				previewLine += lineStr + "\n"
			}
			lastLine = lineStr
			errorCounts++
		}
	}
	if w.anomaly {
		slog.Info("Saving anomalies")
		if err := w.SaveAnomalies(); err != nil {
			return nil, err
		}
		slog.Info("Deleting old anomalies")
		if err := w.DeleteOldAnomalies(); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	matchPercentage := 0.0
	if linesRead > 0 {
		matchPercentage = float64(errorCounts) * 100 / float64(linesRead)
		if matchPercentage > 100 {
			matchPercentage = 100
		}
	}

	// Restrict to two decimal places
	matchPercentage = float64(int(matchPercentage*100)) / 100
	w.lastLineNum = currentLineNum
	w.lastFileSize = bytesRead
	if err := w.saveState(); err != nil {
		if strings.HasPrefix(err.Error(), "database is locked") {
			if err := DeleteDB(w.dbName); err != nil {
				return nil, err
			}
		}
		return nil, err
	}
	return &ScanResult{
		ErrorCount:   errorCounts,
		FirstLine:    firstLine,
		FirstDate:    SearchDate(firstLine),
		PreviewLine:  previewLine,
		LastLine:     lastLine,
		LastDate:     SearchDate(lastLine),
		FilePath:     w.filePath,
		FileInfo:     fileInfo,
		ErrorPercent: matchPercentage,
		LinesRead:    linesRead,
	}, nil
}

func (w *Watcher) loadState() error {
	row := w.db.QueryRow(`SELECT value FROM state WHERE key = ?`, w.lastLineKey)
	var lastLineNum int
	err := row.Scan(&lastLineNum)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	w.lastLineNum = lastLineNum

	row = w.db.QueryRow(`SELECT value FROM state WHERE key = ?`, w.lastFileSizeKey)
	var lastFileSize int64
	err = row.Scan(&lastFileSize)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	w.lastFileSize = lastFileSize
	return nil
}

func (w *Watcher) saveState() error {
	updatedAt := time.Now().Format("2006-01-02 15:04:05")
	_, err := w.db.Exec(`REPLACE INTO state (key, value, updated_at) VALUES (?, ?, ?)`, w.lastLineKey, w.lastLineNum, updatedAt)
	if err != nil {
		return err
	}
	_, err = w.db.Exec(`REPLACE INTO state (key, value, updated_at) VALUES (?, ?, ?)`, w.lastFileSizeKey, w.lastFileSize, updatedAt)
	return err
}

func (w *Watcher) SaveAnomalies() error {
	createdAt := time.Now().Format("2006-01-02 15:04:05")
	for match, value := range w.anomalizer.counter {
		_, err := w.db.Exec(`INSERT INTO anomalies (key, match, value, created_at) VALUES (?, ?, ?, ?)`, w.anomalyKey, match, value, createdAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) DeleteOldAnomalies() error {
	windowAt := time.Now().AddDate(0, 0, -w.anomalyWindow).Format("2006-01-02 15:04:05")
	_, err := w.db.Exec(`DELETE FROM anomalies WHERE key = ? AND created_at < ?`, w.anomalyKey, windowAt)
	return err
}

func (w *Watcher) Close() error {
	// return w.db.Close()
	return nil
}
