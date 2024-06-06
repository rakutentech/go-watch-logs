package pkg

import (
	"os"
	"path/filepath"
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
