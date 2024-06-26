package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"sync"

	"github.com/tidwall/buntdb"
)

type Watcher struct {
	db              *buntdb.DB
	filePath        string
	lastLineKey     string
	lastFileSizeKey string
	matchPattern    string
	ignorePattern   string
	noCache         bool
	lastLineNum     int
	lastFileSize    int64
	mutex           sync.Mutex
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
	// add a suffix to the database name, is just in case some, cuz we are doing os remove
	// and don't want to remove any other file on mis configuration
	if dbName != ":memory:" {
		dbName += ".buntdb"
	}
	db, err := buntdb.Open(dbName)
	if err != nil {
		if err.Error() == "invalid database" {
			slog.Warn("Invalid database, removing and creating a new one", "dbName", dbName)
			err = os.Remove(dbName)
			if err != nil {
				return nil, err
			}
			db, err = buntdb.Open(dbName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
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
func (w *Watcher) NoCache() error {
	return w.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(w.lastLineKey, "0", nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(w.lastFileSizeKey, "0", nil)
		return err
	})
}

type ScanResult struct {
	ErrorCount int
	FirstLine  string
	LastLine   string
}

func (w *Watcher) Scan() (*ScanResult, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
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
	return w.db.View(func(tx *buntdb.Tx) error {
		lastLineStr, err := tx.Get(w.lastLineKey)
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Sscanf(lastLineStr, "%d", &w.lastLineNum) // nolint: errcheck

		lastFileSizeStr, err := tx.Get(w.lastFileSizeKey)
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Sscanf(lastFileSizeStr, "%d", &w.lastFileSize) // nolint: errcheck
		return nil
	})
}

func (w *Watcher) saveState() error {
	return w.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(w.lastLineKey, fmt.Sprintf("%d", w.lastLineNum), nil)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(w.lastFileSizeKey, fmt.Sprintf("%d", w.lastFileSize), nil)
		return err
	})
}

func (w *Watcher) Close() error {
	return w.db.Close()
}
