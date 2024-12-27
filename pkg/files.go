package pkg

import (
	"math"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"
)

func IsTextFile(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Check if the file is empty
	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}
	if fileInfo.Size() == 0 {
		return true, nil
	}

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return false, err
	}

	return utf8.Valid(buffer[:n]), nil
}

func FilesByPattern(pattern string) ([]string, error) {
	// Check if the pattern is a directory
	info, err := os.Stat(pattern)
	if err == nil && info.IsDir() {
		// List all files in the directory
		var files []string
		err := filepath.Walk(pattern, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return files, nil
	}

	// If pattern is not a directory, use Glob to match the pattern
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	return files, nil
}
func GetHomedir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func IsRecentlyModified(fileInfo os.FileInfo, within uint64) bool {
	// Get the current time
	now := time.Now()

	// Get the file's modification time
	modTime := fileInfo.ModTime()

	// Add a 1-hour buffer (3600 seconds) to the "within" duration
	adjustedWithin := within + 3600

	// Ensure that adjustedWithin is within the bounds of int64
	if adjustedWithin > math.MaxInt64 {
		// If the value exceeds the max int64 value, return false as it would cause overflow
		return false
	}

	// Calculate the time difference
	diff := now.Sub(modTime)

	// Check if the difference is within the adjusted duration
	return diff <= time.Duration(adjustedWithin)*time.Second
}
