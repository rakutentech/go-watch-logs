package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
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
	alreadyScanned  int64
}

func NewWatcher(
	dbName string,
	filePath string,
	matchPattern string,
	ignorePattern string,
	noCache bool,
) (*Watcher, error) {
	db, err := buntdb.Open(dbName)
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
		w.lastLineNum = 0
		w.alreadyScanned = 0
	}

	file, err := os.Open(w.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = file.Seek(w.alreadyScanned, 0)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)

	re, err := regexp.Compile(w.matchPattern)
	if err != nil {
		return nil, err
	}
	ri, err := regexp.Compile(w.ignorePattern)
	if err != nil {
		return nil, err
	}

	for scanner.Scan() {
		line := scanner.Text()
		if w.ignorePattern != "" && ri.MatchString(line) {
			w.alreadyScanned += int64(len(line) + 1) // +1 for newline character
			if strings.HasSuffix(line, "\r") {
				w.alreadyScanned++
			}
			continue
		}
		if re.MatchString(line) {
			if firstLine == "" {
				firstLine = line
			}
			lastLine = line
			errorCounts++
		}
		w.alreadyScanned += int64(len(line) + 1) // +1 for newline character
		if strings.HasSuffix(line, "\r") {
			w.alreadyScanned++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	w.lastFileSize = currentFileSize
	if err := w.saveState(); err != nil {
		return nil, err
	}
	return &ScanResult{
		ErrorCount: errorCounts,
		FirstLine:  firstLine,
		LastLine:   lastLine,
	}, nil
}

func (w *Watcher) GetLastLineNum() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.lastLineNum
}

func (w *Watcher) SetLastLineNum(lineNum int) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.lastLineNum = lineNum
	err := w.saveState()
	if err != nil {
		fmt.Println("Error saving state:", err)
	}
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

		alreadyScannedStr, err := tx.Get(w.filePath + "_scanned")
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Sscanf(alreadyScannedStr, "%d", &w.alreadyScanned) // nolint: errcheck

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
		if err != nil {
			return err
		}
		_, _, err = tx.Set(w.filePath+"_scanned", fmt.Sprintf("%d", w.alreadyScanned), nil)
		return err
	})
}

func (w *Watcher) Close() error {
	return w.db.Close()
}
