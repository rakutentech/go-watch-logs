package main

import (
	"flag"
	"fmt"

	"github.com/rakutentech/go-watch-logs/pkg"

	"log/slog"
)

type Flags struct {
	filePath string
	match    string
	dbPath   string
	version  bool
}

var f Flags

var version = "dev"

func main() {
	SetupFlags()
	if f.version {
		slog.Info(version)
		return
	}
	if f.filePath == "" {
		slog.Error("file is required")
		return
	}
	watch()
}

func watch() {
	watcher, err := pkg.NewWatcher(f.dbPath, f.filePath, f.match)
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	errorCount, firstLine, lastLine, err := watcher.ReadFileAndMatchErrors()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Errors: %d\n", errorCount)
	fmt.Printf("%s\n%s", firstLine, lastLine)

	// Get the last line number checked
	lastLineNum := watcher.GetLastLineNum()
	fmt.Println("Last line number checked:", lastLineNum)
}

func SetupFlags() {
	flag.StringVar(&f.filePath, "file-path", "", "path to logs file")
	flag.StringVar(&f.filePath, "f", "", "path to logs file")

	flag.StringVar(&f.dbPath, "db-path", "my.db", "path to db file")
	flag.StringVar(&f.dbPath, "d", "my.db", "path to db file")

	flag.StringVar(&f.match, "match", "", "match string")
	flag.StringVar(&f.match, "m", "", "match string")

	flag.BoolVar(&f.version, "version", false, "print version")
	flag.BoolVar(&f.version, "v", false, "print version")

	flag.Parse()
}
