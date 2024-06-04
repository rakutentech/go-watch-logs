package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gookit/color"

	n "github.com/rakutentech/go-alertnotification"
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
	color.Success.Println(watcher.GetLastLineNum())

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
	color.Success.Println(watcher.GetLastLineNum())

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
		color.Secondary.Println("Sending to Teams.............")
		fmt.Println(teamsMsg)
		alert := n.NewAlert(fmt.Errorf(teamsMsg), nil)
		if err := alert.Notify(); err != nil {
			color.Danger.Println(err)
		}
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
	SetMSTeams()
}

func SetMSTeams() {
	hostname, _ := os.Hostname()
	os.Setenv("APP_NAME", f.filePath)
	os.Setenv("APP_ENV", hostname)
	os.Setenv("MS_TEAMS_ALERT_ENABLED", "true")
	os.Setenv("MS_TEAMS_WEBHOOK", f.msTeamsHook)
	os.Setenv("MS_TEAMS_CARD_SUBJECT", fmt.Sprintf("match: <code>%s</code><br>ignore: <code>%s</code>", f.match, f.ignore))
	os.Setenv("ALERT_CARD_SUBJECT", "GO-WATCH-LOGS")
	proxyVars := []string{"https_proxy", "http_proxy", "HTTPS_PROXY", "HTTP_PROXY"}

	for _, proxyVar := range proxyVars {
		if os.Getenv(proxyVar) != "" {
			os.Setenv("MS_TEAMS_PROXY_URL", os.Getenv(proxyVar))
			break
		}
	}
}
