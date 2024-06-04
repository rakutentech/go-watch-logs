package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gookit/color"
	gmt "github.com/kevincobain2000/go-msteams/src"

	"github.com/rakutentech/go-watch-logs/pkg"
)

type Flags struct {
	filePath    string
	match       string
	ignore      string
	dbPath      string
	minError    int
	msTeamsHook string
	version     bool
}

var f Flags

var version = "dev"

func main() {
	SetupFlags()
	if f.version {
		color.Secondary.Println(version)
		return
	}
	if f.filePath == "" {
		color.Danger.Println("file-path is required")
		return
	}
	// check if file exists
	if _, err := os.Stat(f.filePath); os.IsNotExist(err) {
		color.Danger.Println("file does not exist")
		return
	}
	watch()
}

func watch() {
	watcher, err := pkg.NewWatcher(f.dbPath, f.filePath, f.match, f.ignore)
	if err != nil {
		color.Danger.Println(err)
		return
	}
	defer watcher.Close()

	color.Secondary.Print("1st line no..................")
	color.Cyan.Println(watcher.GetLastLineNum())

	errorCount, firstLine, lastLine, err := watcher.ReadFileAndMatchErrors()
	if err != nil {
		color.Danger.Println(err)
		return
	}
	color.Secondary.Print("error count..................")
	color.Danger.Println(errorCount)

	// first line
	color.Secondary.Print("1st line.....................")
	fmt.Println(firstLine)

	color.Secondary.Print("last line....................")
	fmt.Println(lastLine)

	color.Secondary.Print("last line no.................")
	color.Cyan.Println(watcher.GetLastLineNum())

	if errorCount < 0 {
		return
	}
	if errorCount < f.minError {
		return
	}
	notify(errorCount, firstLine, lastLine)
}

func notify(errorCount int, firstLine, lastLine string) {
	if f.msTeamsHook != "" {
		teamsMsg := fmt.Sprintf("total errors: %d\n\n", errorCount)
		teamsMsg += fmt.Sprintf("1st error<pre>\n\n%s</pre>\n\nlast error<pre>\n\n%s</pre>", firstLine, lastLine)
		color.Secondary.Print("Sending to Teams.............")
		color.Warn.Println("Work in Progress")

		hostname, _ := os.Hostname()
		subject := fmt.Sprintf("match: <code>%s</code>", f.match)
		subject += "<br>"
		subject += fmt.Sprintf("ignore: <code>%s</code>", f.ignore)
		err := gmt.Send(hostname, f.filePath, subject, "red", teamsMsg, f.msTeamsHook, proxy())
		if err != nil {
			color.Danger.Println(err)
		}
		color.Secondary.Print("Sent to Teams................")
		color.Success.Println("Done")
	}
}

func SetupFlags() {
	flag.StringVar(&f.filePath, "file-path", "", "full path to the log file")
	flag.StringVar(&f.dbPath, "db-path", ".go-watch-logs.db", "path to store db file")
	flag.StringVar(&f.match, "match", "", "regex for matching errors (empty to match all lines)")
	flag.StringVar(&f.ignore, "ignore", "", "regex for ignoring errors (empty to ignore none)")
	flag.IntVar(&f.minError, "min-error", 1, "on minimum error threshold to notify")
	flag.BoolVar(&f.version, "version", false, "")

	flag.StringVar(&f.msTeamsHook, "ms-teams-hook", "", "ms teams webhook")

	flag.Parse()
}

func proxy() string {
	proxyVars := []string{"https_proxy", "http_proxy", "HTTPS_PROXY", "HTTP_PROXY"}

	for _, proxyVar := range proxyVars {
		if os.Getenv(proxyVar) != "" {
			return os.Getenv(proxyVar)
		}
	}
	return ""
}
