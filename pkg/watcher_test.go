package pkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
	filePath := "test.log"    // nolint: goconst
	matchPattern := "error:1" // nolint: goconst
	ignorePattern := "ignore" // nolint: goconst

	f := Flags{
		DBPath:            "test",
		Anomaly:           false,
		AnomalyWindowDays: 1,
		Match:             matchPattern,
		Ignore:            ignorePattern,
	}

	watcher, err := NewWatcher(filePath, f)
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

	matchPattern := `error:1` // nolint: goconst
	ignorePattern := `ignore` // nolint: goconst

	f := Flags{
		DBPath:            "test",
		Anomaly:           false,
		AnomalyWindowDays: 1,
		Match:             matchPattern,
		Ignore:            ignorePattern,
	}

	watcher, err := NewWatcher(filePath, f)
	assert.NoError(t, err)
	defer watcher.Close()

	result, err := watcher.Scan()
	assert.NoError(t, err)
	assert.Equal(t, 2, result.ErrorCount)
	assert.Equal(t, "error:1", result.FirstLine)
	assert.Equal(t, "error:1", result.LastLine)
}

func TestLogRotation(t *testing.T) {
	content := `line1
error:1
line2`
	filePath, err := setupTempFile(content)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	matchPattern := `error:1` // nolint: goconst
	ignorePattern := `ignore` // nolint: goconst

	f := Flags{
		DBPath:            "test",
		Anomaly:           false,
		AnomalyWindowDays: 1,
		Match:             matchPattern,
		Ignore:            ignorePattern,
	}

	watcher, err := NewWatcher(filePath, f)
	assert.NoError(t, err)
	defer watcher.Close()

	result, err := watcher.Scan()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ErrorCount)
	assert.Equal(t, "error:1", result.FirstLine)
	assert.Equal(t, "error:1", result.LastLine)

	// Simulate log rotation by truncating the file
	err = os.WriteFile(filePath, []byte("error:1\n"), 0644) // nolint: gosec
	assert.NoError(t, err)

	result, err = watcher.Scan()
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ErrorCount)
	assert.Equal(t, "error:1", result.FirstLine)
	assert.Equal(t, "error:1", result.LastLine)
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

	matchPattern := `error:1`
	ignorePattern := `ignore`

	f := Flags{
		DBPath:            "test",
		Anomaly:           false,
		AnomalyWindowDays: 1,
		Match:             matchPattern,
		Ignore:            ignorePattern,
	}

	watcher, err := NewWatcher(filePath, f)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	for i := 0; i < b.N; i++ {
		_, err := watcher.Scan()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadAndSaveState(b *testing.B) {
	dbName := "test.db"
	filePath := "test.log"
	matchPattern := "error:1"
	ignorePattern := "ignore"

	f := Flags{
		DBPath:            dbName,
		Anomaly:           false,
		AnomalyWindowDays: 1,
		Match:             matchPattern,
		Ignore:            ignorePattern,
	}

	watcher, err := NewWatcher(filePath, f)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	watcher.lastLineNum = 10

	for i := 0; i < b.N; i++ {
		_, err := NewWatcher(filePath, f)
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

	matchPattern := `error:1`
	ignorePattern := `ignore`

	f := Flags{
		DBPath:            "test",
		Anomaly:           false,
		AnomalyWindowDays: 1,
		Match:             matchPattern,
		Ignore:            ignorePattern,
	}

	watcher, err := NewWatcher(filePath, f)
	if err != nil {
		b.Fatal(err)
	}
	defer watcher.Close()

	result, err := watcher.Scan()
	if err != nil {
		b.Fatal(err)
	}
	if result.ErrorCount != 1 || result.FirstLine != "error:1" || result.LastLine != "error:1" {
		b.Fatalf("Unexpected results: count=%d, first=%s, last=%s", result.ErrorCount, result.FirstLine, result.LastLine)
	}

	err = os.WriteFile(filePath, []byte("error:1\n"), 0644) // nolint: gosec
	if err != nil {
		b.Fatal(err)
	}

	watcher.lastLineNum = 0

	for i := 0; i < b.N; i++ {
		_, err := watcher.Scan()
		if err != nil {
			b.Fatal(err)
		}
	}
}
