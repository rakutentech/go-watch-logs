package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/jasonlvhit/gocron"
	gmt "github.com/kevincobain2000/go-msteams/src"

	"github.com/rakutentech/go-watch-logs/pkg"
)

type Flags struct {
	filePath string
	match    string
	ignore   string
	dbPath   string

	minError    int
	every       uint64
	proxy       string
	msTeamsHook string
	noCache     bool
	version     bool
}

var f Flags

var version = "dev"

func main() {
	SetupFlags()
	if err := validate(); err != nil {
		color.Danger.Println(err)
		return
	}
	filePaths, err := pkg.FilesByPattern(f.filePath)
	if err != nil {
		color.Danger.Println(err)
		return
	}
	if len(filePaths) == 0 {
		color.Danger.Println("no files found", f.filePath)
		return
	}
	for _, filePath := range filePaths {
		isText, err := pkg.IsTextFile(filePath)
		if err != nil {
			color.Danger.Println(err)
			return
		}
		if !isText {
			color.Danger.Println("file is not a text file", filePath)
			return
		}
	}

	for _, filePath := range filePaths {
		watch(filePath)
	}
	if f.every > 0 {
		cron(filePaths)
	}
}

func cron(filePaths []string) {
	for _, filePath := range filePaths {
		if err := gocron.Every(f.every).Second().Do(watch, filePath); err != nil {
			color.Danger.Println(err)
			return
		}
	}
	<-gocron.Start()
}

func validate() error {
	if f.version {
		color.Secondary.Println(version)
		return fmt.Errorf("version")
	}
	if f.filePath == "" {
		color.Danger.Println("file-path is required")
		return fmt.Errorf("filepath")
	}
	return nil
}

func watch(filePath string) {
	watcher, err := pkg.NewWatcher(f.dbPath, filePath, f.match, f.ignore, f.noCache)
	if err != nil {
		color.Danger.Println(err)
		return
	}
	defer watcher.Close()

	color.Secondary.Print("scanning..................... ")
	fmt.Println(filePath)

	color.Secondary.Print("1st line no.................. ")
	fmt.Println(watcher.GetLastLineNum())

	errorCount, firstLine, lastLine, err := watcher.ReadFileAndMatchErrors()
	if err != nil {
		color.Danger.Println(err)
		return
	}
	color.Secondary.Print("error count.................. ")
	color.Danger.Println(errorCount)

	// first line
	color.Secondary.Print("1st line..................... ")
	fmt.Println(pkg.Truncate(firstLine, 50))

	color.Secondary.Print("last line.................... ")
	fmt.Println(pkg.Truncate(lastLine, 50))

	color.Secondary.Print("last line no................. ")
	fmt.Println(watcher.GetLastLineNum())

	fmt.Println()

	if errorCount < 0 {
		return
	}
	if errorCount < f.minError {
		return
	}
	notify(errorCount, watcher.GetLastLineNum(), firstLine, lastLine)
}

func notify(errorCount, lastLineNum int, firstLine, lastLine string) {
	if f.msTeamsHook != "" {
		teamsMsg := fmt.Sprintf("total errors: %d\n\n", errorCount)
		teamsMsg += fmt.Sprintf("1st error<pre>\n\n%s</pre>\n\nlast error<pre>\n\n%s</pre>", firstLine, lastLine)
		color.Secondary.Print("Sending to Teams.............")
		color.Warn.Println("Work in Progress")

		hostname, _ := os.Hostname()
		subject := fmt.Sprintf("match: <code>%s</code>", f.match)
		subject += "<br>"
		subject += fmt.Sprintf("ignore: <code>%s</code>", f.ignore)
		subject += "<br>"
		subject += fmt.Sprintf("line no: <code>%d</code>", lastLineNum)
		err := gmt.Send(hostname, f.filePath, subject, "red", teamsMsg, f.msTeamsHook, proxy())
		if err != nil {
			color.Danger.Println(err)
		}
		color.Secondary.Print("Sent to Teams................ ")
		color.Success.Println("Done")
	}
}

func SetupFlags() {
	flag.StringVar(&f.filePath, "file-path", "", "full path to the log file")
	flag.StringVar(&f.dbPath, "db-path", ".go-watch-logs.db", "path to store db file")
	flag.StringVar(&f.match, "match", "", "regex for matching errors (empty to match all lines)")
	flag.StringVar(&f.ignore, "ignore", "", "regex for ignoring errors (empty to ignore none)")
	flag.Uint64Var(&f.every, "every", 0, "run every n seconds (0 to run once)")
	flag.IntVar(&f.minError, "min-error", 1, "on minimum num of errors should notify")
	flag.BoolVar(&f.noCache, "no-cache", false, "read back from the start of the file (default false)")
	flag.BoolVar(&f.version, "version", false, "")

	flag.StringVar(&f.proxy, "proxy", "", "http proxy for webhooks")
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
	return f.proxy
}
