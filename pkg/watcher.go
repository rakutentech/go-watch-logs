package pkg

import (
	"bufio"
	"errors"
	"fmt"
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
	lastLineNum     int
	lastFileSize    int64
	mutex           sync.Mutex
}

func NewWatcher(dbName string, filePath string, matchPattern string) (*Watcher, error) {
	db, err := buntdb.Open(dbName)
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		db:              db,
		filePath:        filePath,
		matchPattern:    matchPattern,
		lastLineKey:     Hash(filePath + "llk"),
		lastFileSizeKey: Hash(filePath + "llks"),
	}

	err = watcher.loadState()
	if err != nil {
		return nil, err
	}

	return watcher, nil
}

func (w *Watcher) ReadFileAndMatchErrors() (int, string, string, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	errorCounts := 0
	firstLine := ""
	lastLine := ""

	fileInfo, err := os.Stat(w.filePath)
	if err != nil {
		return errorCounts, firstLine, lastLine, err
	}

	currentFileSize := fileInfo.Size()

	// Detect log rotation
	if currentFileSize < w.lastFileSize {
		w.lastLineNum = 0
	}

	file, err := os.Open(w.filePath)
	if err != nil {
		return errorCounts, firstLine, lastLine, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	re, err := regexp.Compile(w.matchPattern)
	if err != nil {
		return errorCounts, firstLine, lastLine, err
	}

	currentLineNum := 0
	for scanner.Scan() {
		currentLineNum++
		if currentLineNum <= w.lastLineNum {
			continue
		}
		line := scanner.Text()
		if re.MatchString(line) {
			if firstLine == "" {
				firstLine = line
			}
			lastLine = line
			errorCounts++
		}
	}

	if err := scanner.Err(); err != nil {
		return errorCounts, firstLine, lastLine, err
	}

	w.lastLineNum = currentLineNum
	w.lastFileSize = currentFileSize
	if err := w.saveState(); err != nil {
		return errorCounts, firstLine, lastLine, err
	}
	return errorCounts, firstLine, lastLine, nil
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
		fmt.Sscanf(lastLineStr, "%d", &w.lastLineNum)

		lastFileSizeStr, err := tx.Get(w.lastFileSizeKey)
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Sscanf(lastFileSizeStr, "%d", &w.lastFileSize)
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
