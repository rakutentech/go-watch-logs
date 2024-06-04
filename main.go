package main

import (
	"flag"
	"fmt"
	"os"

	n "github.com/rakutentech/go-alertnotification"
	"github.com/rakutentech/go-watch-logs/pkg"
)

type Flags struct {
	filePath    string
	match       string
	ignore      string
	dbPath      string
	msTeamsHook string
	version     bool
}

var f Flags

var version = "dev"

func main() {
	SetupFlags()
	SetMSTeams()
	if f.version {
		fmt.Println(version)
		return
	}
	if f.filePath == "" {
		fmt.Println("file is required")
		return
	}
	watch()
}

func watch() {
	watcher, err := pkg.NewWatcher(f.dbPath, f.filePath, f.match, f.ignore)
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

	if errorCount > 0 && f.msTeamsHook != "" {
		teamsMsg := fmt.Sprintf("total errors: %d\n\n", errorCount)
		teamsMsg += fmt.Sprintf("1st error<pre>\n\n%s</pre>\n\nlast error<pre>\n\n%s</pre>", firstLine, lastLine)
		fmt.Println("ms teams message:")
		fmt.Println(teamsMsg)
		alert := n.NewAlert(fmt.Errorf(teamsMsg), nil)
		go func() {
			if err := alert.Notify(); err != nil {
				fmt.Println("error sending alert:", err)
			}
		}()
	}

	fmt.Printf("error count: %d\n", errorCount)
	fmt.Printf("last line number: %d\n", watcher.GetLastLineNum())
}

func SetupFlags() {
	flag.StringVar(&f.filePath, "file-path", "", "path to logs file")
	flag.StringVar(&f.dbPath, "db-path", "go-watch-logs.db", "path to db file")
	flag.StringVar(&f.match, "match", "", "match pattern")
	flag.StringVar(&f.ignore, "ignore", "", "ignore pattern")
	flag.BoolVar(&f.version, "version", false, "print version")

	flag.StringVar(&f.msTeamsHook, "ms-teams-hook", "", "ms teams webhook")

	flag.Parse()
}

func SetMSTeams() {
	if f.msTeamsHook == "" {
		return
	}

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
