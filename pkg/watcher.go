package pkg

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

type Watcher struct {
	db              *sql.DB
	dbName          string // full path
	filePath        string
	lastLineKey     string
	lastFileSizeKey string
	matchPattern    string
	ignorePattern   string
	maxBufferSizeMB int
	lastLineNum     int
	lastFileSize    int64
	timestampNow    string
}

func NewWatcher(
	filePath string,
	f Flags,
) (*Watcher, error) {
	suffix := Hash(fmt.Sprintf("%s-%s-%s-%s-%d", f.FilePath, f.Match, f.Ignore, f.MSTeamsHook, f.Every)) + ".sqlite"
	dbName := f.DBPath + "." + suffix
	db, err := InitDB(dbName)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	watcher := &Watcher{
		db:              db,
		dbName:          dbName,
		filePath:        filePath,
		matchPattern:    f.Match,
		ignorePattern:   f.Ignore,
		lastLineKey:     "llk-" + filePath,
		lastFileSizeKey: "llks-" + filePath,
		timestampNow:    now.Format("2006-01-02 15:04:05"),
		maxBufferSizeMB: f.MaxBufferSizeMB,
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
	if w.maxBufferSizeMB > 0 {
		// for large lines
		scanner.Buffer(make([]byte, 0, 64*1024), w.maxBufferSizeMB*1024*1024)
	}
	currentLineNum := 1
	linesRead := 0
	bytesRead := w.lastFileSize

	for scanner.Scan() {
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // Adding 1 for the newline character
		currentLineNum++
		linesRead = currentLineNum - w.lastLineNum
		// convert to positive number
		if linesRead < 0 {
			linesRead = -linesRead
		}

		if w.ignorePattern != "" && regIgnore.Match(line) {
			continue
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
		ErrorCount:   matchCounts,
		FirstDate:    SearchDate(firstLine),
		LastDate:     SearchDate(lastLine),
		FirstLine:    firstLine,
		PreviewLine:  previewLine,
		LastLine:     lastLine,
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
	_, err := w.db.Exec(`REPLACE INTO state (key, value, updated_at) VALUES (?, ?, ?)`, w.lastLineKey, w.lastLineNum, w.timestampNow)
	if err != nil {
		return err
	}
	_, err = w.db.Exec(`REPLACE INTO state (key, value, updated_at) VALUES (?, ?, ?)`, w.lastFileSizeKey, w.lastFileSize, w.timestampNow)
	return err
}

func (w *Watcher) Close() error {
	return w.db.Close()
}
