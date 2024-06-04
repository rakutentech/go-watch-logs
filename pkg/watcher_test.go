package pkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	inMemory = ":memory:"
)

func setupTempFile(content string) (string, error) {
	tmpfile, err := os.CreateTemp("", "test.log")
	if err != nil {
		return "", err
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

func TestNewWatcher(t *testing.T) {
	dbName := inMemory
	filePath := "test.log"    // nolint: goconst
	matchPattern := "error:1" // nolint: goconst
	ignorePattern := "ignore" // nolint: goconst

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	assert.NoError(t, err)
	assert.NotNil(t, watcher)

	defer watcher.Close()
}

func TestReadFileAndMatchErrors(t *testing.T) {
	content := `line1
error:1
error:2
line2
error:1`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	dbName := inMemory
	matchPattern := `error:1` // nolint: goconst
	ignorePattern := `ignore` // nolint: goconst

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	assert.NoError(t, err)
	defer watcher.Close()

	count, first, last, err := watcher.ReadFileAndMatchErrors()
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, "error:1", first)
	assert.Equal(t, "error:1", last)
}

func TestSetAndGetLastLineNum(t *testing.T) {
	dbName := inMemory
	filePath := "test.log"
	matchPattern := "error:1"
	ignorePattern := "ignore"

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	assert.NoError(t, err)
	defer watcher.Close()

	watcher.SetLastLineNum(10)
	lineNum := watcher.GetLastLineNum()
	assert.Equal(t, 10, lineNum)
}

func TestLoadAndSaveState(t *testing.T) {
	dbName := "test.db"
	filePath := "test.log"
	matchPattern := "error:1" // nolint: goconst
	ignorePattern := "ignore" // nolint: goconst

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	assert.NoError(t, err)
	defer watcher.Close()

	watcher.SetLastLineNum(10)

	newWatcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	assert.NoError(t, err)
	defer newWatcher.Close()

	lineNum := newWatcher.GetLastLineNum()
	assert.Equal(t, 10, lineNum)

	os.Remove("test.db")
}

func TestLogRotation(t *testing.T) {
	content := `line1
error:1
line2`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	dbName := inMemory
	matchPattern := `error:1` // nolint: goconst
	ignorePattern := `ignore` // nolint: goconst

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	assert.NoError(t, err)
	defer watcher.Close()

	count, first, last, err := watcher.ReadFileAndMatchErrors()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, "error:1", first)
	assert.Equal(t, "error:1", last)

	// Simulate log rotation by truncating the file
	err = os.WriteFile(filePath, []byte("new content\nerror:1\n"), 0644) // nolint: gosec
	assert.NoError(t, err)

	// Ensure Watcher detects log rotation
	watcher.SetLastLineNum(0) // Reset line number to simulate log rotation

	count, first, last, err = watcher.ReadFileAndMatchErrors()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, "error:1", first)
	assert.Equal(t, "error:1", last)
}

func BenchmarkReadFileAndMatchErrors(b *testing.B) {
	content := `line1
error:1
error:2
line2
error:1`
	filePath, err := setupTempFile(content)
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(filePath)

	dbName := inMemory
	matchPattern := `error:1`
	ignorePattern := `ignore`

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	for i := 0; i < b.N; i++ {
		_, _, _, err := watcher.ReadFileAndMatchErrors()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetAndGetLastLineNum(b *testing.B) {
	dbName := inMemory
	filePath := "test.log"
	matchPattern := "error:1"
	ignorePattern := "ignore"

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	for i := 0; i < b.N; i++ {
		watcher.SetLastLineNum(10)
		_ = watcher.GetLastLineNum()
	}
}

func BenchmarkLoadAndSaveState(b *testing.B) {
	dbName := "test.db"
	filePath := "test.log"
	matchPattern := "error:1"
	ignorePattern := "ignore"

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	watcher.SetLastLineNum(10)

	for i := 0; i < b.N; i++ {
		_, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLogRotation(b *testing.B) {
	content := `line1
error:1
line2`
	filePath, err := setupTempFile(content)
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(filePath)

	dbName := inMemory
	matchPattern := `error:1`
	ignorePattern := `ignore`

	watcher, err := NewWatcher(dbName, filePath, matchPattern, ignorePattern)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	count, first, last, err := watcher.ReadFileAndMatchErrors()
	if err != nil {
		b.Fatal(err)
	}
	if count != 1 || first != "error:1" || last != "error:1" {
		b.Fatalf("Unexpected results: count=%d, first=%s, last=%s", count, first, last)
	}

	err = os.WriteFile(filePath, []byte("new content\nerror:1\n"), 0644) // nolint: gosec
	if err != nil {
		b.Fatal(err)
	}

	watcher.SetLastLineNum(0)

	for i := 0; i < b.N; i++ {
		_, _, _, err := watcher.ReadFileAndMatchErrors()
		if err != nil {
			b.Fatal(err)
		}
	}
}
