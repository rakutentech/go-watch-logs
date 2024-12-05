package pkg

import (
	"bufio"
	"database/sql"
	"io"
	"os"
	"regexp"
	"strings"
)

type Watcher struct {
	db              *sql.DB
	dbName          string // full path
	filePath        string
	lastLineKey     string
	lastFileSizeKey string
	matchPattern    string
	ignorePattern   string
	lastLineNum     int
	lastFileSize    int64
}

func NewWatcher(
	dbName string,
	filePath string,
	matchPattern string,
	ignorePattern string,
) (*Watcher, error) {
	dbName += ".sqlite"
	db, err := InitDB(dbName)
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		db:              db,
		dbName:          dbName,
		filePath:        filePath,
		matchPattern:    matchPattern,
		ignorePattern:   ignorePattern,
		lastLineKey:     "llk-" + filePath,
		lastFileSizeKey: "llks-" + filePath,
	}
	if err := watcher.loadState(); err != nil {
		return nil, err
	}

	return watcher, nil
}

type ScanResult struct {
	FilePath    string
	ErrorCount  int
	FirstLine   string
	FirstDate   string
	PreviewLine string
	LastLine    string
	LastDate    string
}

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

	re, err := regexp.Compile(w.matchPattern)
	if err != nil {
		return nil, err
	}
	ri, err := regexp.Compile(w.ignorePattern)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	currentLineNum := 1
	bytesRead := w.lastFileSize

	for scanner.Scan() {
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // Adding 1 for the newline character
		currentLineNum++
		if w.ignorePattern != "" && ri.Match(line) {
			continue
		}
		if re.Match(line) {
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

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	w.lastLineNum = currentLineNum
	w.lastFileSize = bytesRead
	if err := w.saveState(); err != nil {
		if strings.HasPrefix(err.Error(), "database is locked") {
			if err := Vacuum(w.dbName); err != nil {
				return nil, err
			}
		}
		return nil, err
	}
	return &ScanResult{
		ErrorCount:  errorCounts,
		FirstLine:   firstLine,
		FirstDate:   SearchDate(firstLine),
		PreviewLine: previewLine,
		LastLine:    lastLine,
		LastDate:    SearchDate(lastLine),
		FilePath:    w.filePath,
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
	_, err := w.db.Exec(`REPLACE INTO state (key, value) VALUES (?, ?)`, w.lastLineKey, w.lastLineNum)
	if err != nil {
		return err
	}
	_, err = w.db.Exec(`REPLACE INTO state (key, value) VALUES (?, ?)`, w.lastFileSizeKey, w.lastFileSize)
	return err
}

func (w *Watcher) Close() error {
	return w.db.Close()
}
