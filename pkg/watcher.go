package pkg

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	_ "github.com/mattn/go-sqlite3" // nolint: revive
)

type Watcher struct {
	db              *sql.DB
	filePath        string
	lastLineKey     string
	lastFileSizeKey string
	matchPattern    string
	ignorePattern   string
	noCache         bool
	lastLineNum     int
	lastFileSize    int64
}

func NewWatcher(
	dbName string,
	filePath string,
	matchPattern string,
	ignorePattern string,
	noCache bool,
) (*Watcher, error) {
	if dbName == "" {
		return nil, errors.New("dbName is required")
	}

	dbName += ".sqlite"

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		db:              db,
		filePath:        filePath,
		matchPattern:    matchPattern,
		ignorePattern:   ignorePattern,
		noCache:         noCache,
		lastLineKey:     Hash(filePath + "llk"),
		lastFileSizeKey: Hash(filePath + "llks"),
	}
	if err := watcher.CreateTableIfNotExists(); err != nil {
		return nil, err
	}

	if watcher.noCache {
		if err := watcher.NoCache(); err != nil {
			return nil, err
		}
	}

	if err := watcher.loadState(); err != nil {
		return nil, err
	}

	return watcher, nil
}

func (w *Watcher) CreateTableIfNotExists() error {
	// Create table if not exists
	_, err := w.db.Exec(`
	CREATE TABLE IF NOT EXISTS watcher_state (
		key TEXT PRIMARY KEY,
		value TEXT
	)`)
	return err
}
func (w *Watcher) NoCache() error {
	_, err := w.db.Exec("INSERT OR REPLACE INTO watcher_state (key, value) VALUES (?, ?), (?, ?)",
		w.lastLineKey, "0",
		w.lastFileSizeKey, "0")
	return err
}

type ScanResult struct {
	ErrorCount int
	FirstLine  string
	LastLine   string
}

func (w *Watcher) Scan() (*ScanResult, error) {
	errorCounts := 0
	firstLine := ""
	lastLine := ""

	fileInfo, err := os.Stat(w.filePath)
	if err != nil {
		return nil, err
	}

	currentFileSize := fileInfo.Size()

	// Detect log rotation
	if currentFileSize < w.lastFileSize {
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
	currentLineNum := w.lastLineNum
	bytesRead := w.lastFileSize

	for scanner.Scan() {
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // Adding 1 for the newline character
		currentLineNum++
		if w.ignorePattern != "" && ri.Match(line) {
			continue
		}
		if re.Match(line) {
			if firstLine == "" {
				firstLine = string(line)
			}
			lastLine = string(line)
			errorCounts++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	w.lastLineNum = currentLineNum
	w.lastFileSize = bytesRead
	if err := w.saveState(); err != nil {
		return nil, err
	}
	return &ScanResult{
		ErrorCount: errorCounts,
		FirstLine:  firstLine,
		LastLine:   lastLine,
	}, nil
}

func (w *Watcher) loadState() error {
	var lastLineStr, lastFileSizeStr string

	err := w.db.QueryRow("SELECT value FROM watcher_state WHERE key = ?", w.lastLineKey).Scan(&lastLineStr)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	fmt.Sscanf(lastLineStr, "%d", &w.lastLineNum) // nolint: errcheck

	err = w.db.QueryRow("SELECT value FROM watcher_state WHERE key = ?", w.lastFileSizeKey).Scan(&lastFileSizeStr)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	fmt.Sscanf(lastFileSizeStr, "%d", &w.lastFileSize) // nolint: errcheck

	return nil
}

func (w *Watcher) saveState() error {
	_, err := w.db.Exec("INSERT OR REPLACE INTO watcher_state (key, value) VALUES (?, ?), (?, ?)",
		w.lastLineKey, fmt.Sprintf("%d", w.lastLineNum),
		w.lastFileSizeKey, fmt.Sprintf("%d", w.lastFileSize))
	return err
}

func (w *Watcher) Close() error {
	return w.db.Close()
}
