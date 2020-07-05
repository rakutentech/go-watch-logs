package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/hpcloud/tail"
	notifier "github.com/rakutentech/go-alertnotification"
)

// WatchLogsImplementation interface
type WatchLogsImplementation interface {
	AutoRecover(fn func(), watchFile *string, regexps []string, limit *int, counter map[string]int, recoveryCmd *string) (recovered interface{})
	StartWatcher(watchFile *string, regexps *[]string, limit *int, counter *map[string]int, recoveryCmd *string)
	Shellout(command string) (string, string, error)
	StartCounter(counter *map[string]int, regexps *[]string, seconds *int)
	InitCounter(counter *map[string]int, regexps *[]string, limit *int)
	IncrementCounter(counter *map[string]int, re string)
	GetMatchedRegexp(str string, regexps *[]string) string
	Notify(msg string)
	SetInterval(action func(), interval time.Duration) func()
}

// WatchLogs struct
type WatchLogs struct {
	WatchLogsImplementation
}

var w WatchLogs

// Usage of cli
func Usage() string {
	msg := `
Version v1.0.0
--help
        Prints this Usage
--limit int
        Limit to notify (default 10)
--seconds int
        Limit notify per number of second (default 30)
--watch-file string
        Path to the file to tail
-recovery-cmd string
        Shell cmd to execute on match found (default "")

Basic Usage:
        go-watch-logs --limit=10 --seconds=30 --watch-file=/path/to/your.log  "regexp1" "regexp2"
Description:
        Will send max 10 notifications every 30 seconds for regexp1 matched per line in your.log file
		And will send max 10 notifications every 30 seconds for regexp2 matched per line in your.log file
Examples:
    1) Regexps complex
        go-watch-logs --limit=10 --seconds=30 --watch-file=/path/to/your.log  "Traceback" "^Error|^error" "^[Error]"
        go-watch-logs --limit=10 --seconds=30 --watch-file=/path/to/your.log  "Traceback" "Error|error" "^[Error]|[ERROR]"
`
	return msg
}

func main() {
	help := flag.Bool("help", false, "Prints Usage")
	limit := flag.Int("limit", 10, "Limit to notify")
	seconds := flag.Int("seconds", 30, "Limit notify per number of second")
	watchFile := flag.String("watch-file", "", "Path to the file to tail")
	recoveryCmd := flag.String("recovery-cmd", "", "Shell cmd to execute on recovery")

	flag.Parse()
	regexps := flag.Args()

	if *help == true {
		fmt.Println(Usage())
		os.Exit(2)
	}

	if *watchFile == "" {
		fmt.Println(Usage())
		os.Exit(2)
	}

	fmt.Printf(`
All good! Watch started.
-------
Watch file:%s
Seconds:%d
Limit:%d
Regexps:%v
-------
`, *watchFile, *seconds, *limit, flag.Args())

	// @var counter keeps the ledger of count of num of notifications sent for a regexp within a num of seconds
	// @var counter is reset after num of seconds have passed
	counter := map[string]int{}

	w.InitCounter(&counter, &regexps, limit)
	w.StartCounter(&counter, &regexps, seconds)

	w.AutoRecover(func() {
		w.StartWatcher(watchFile, &regexps, limit, &counter, recoveryCmd)
	}, watchFile, regexps, limit, counter, recoveryCmd)
}

// AutoRecover recursively the function that panics
// tail will panic when the file is deleted, or temporarily deleted
// this function allows this cli to never exit
func (w *WatchLogs) AutoRecover(fn func(), watchFile *string, regexps []string, limit *int, counter map[string]int, recoveryCmd *string) (recovered interface{}) {
	defer func() {
		recovered = recover()
		fmt.Println("Auto recover")
		w.AutoRecover(func() {
			w.StartWatcher(watchFile, &regexps, limit, &counter, recoveryCmd)
		}, watchFile, regexps, limit, counter, recoveryCmd)
	}()
	fn()
	return
}

// StartWatcher to tail the file
// Read each line and match regexp provided
// Upon a match sends notification based on the .env values
// Limits sending notifications by num of notifications sent
func (w *WatchLogs) StartWatcher(watchFile *string, regexps *[]string, limit *int, counter *map[string]int, recoveryCmd *string) {
	t, _ := tail.TailFile(*watchFile, tail.Config{Follow: true})

	for line := range t.Lines {
		if re := w.GetMatchedRegexp(line.Text, regexps); re != "" {
			if (*counter)[re] < *limit {
				w.IncrementCounter(counter, re)
				fmt.Println("Will notify: " + line.Text)
				go w.Notify(line.Text)
				go w.Shellout(*recoveryCmd)
			}
		}
	}
}

// Shellout .. call shell and return results
// Example: Shellout("sudo systemctl restart httpd")
func (w *WatchLogs) Shellout(command string) (string, string, error) {
	if command == "" {
		return "", "", nil
	}
	fmt.Println("Will execute cmd: " + command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// StartCounter to reset number of notifications sent for given num of seconds
// @var counter is reset after num of seconds have passed
func (w *WatchLogs) StartCounter(counter *map[string]int, regexps *[]string, seconds *int) {
	fmt.Println("Start counter")
	w.SetInterval(func() {
		fmt.Println("Reset counter")
		for _, re := range *regexps {
			(*counter)[re] = 0
		}
	}, time.Duration(*seconds)*time.Second)
}

// InitCounter acts as a delay, as for the first time when cli is executed, existing tail is read
// We don't want to send notifications for errors that has happened prior to the execution of this cli
func (w *WatchLogs) InitCounter(counter *map[string]int, regexps *[]string, limit *int) {
	fmt.Println("Init counter")
	for _, re := range *regexps {
		(*counter)[re] = *limit
	}
}

// IncrementCounter for the ledger to limit errors for same regexp
func (w *WatchLogs) IncrementCounter(counter *map[string]int, re string) {
	(*counter)[re]++
	for regex, count := range *counter {
		fmt.Printf("Regexp:%s, count:%d\n", regex, count)
	}
}

// GetMatchedRegexp if str matches any of the regexps
// returns the matched regexp
func (w *WatchLogs) GetMatchedRegexp(str string, regexps *[]string) string {
	for _, re := range *regexps {
		match, err := regexp.MatchString(re, str)
		if err != nil {
			fmt.Printf("Regexp: %s is invalid", re)
		}
		if match == true {
			return re
		}
	}
	return ""
}

// Notify notifies alert based on .env vars
// Also see: https://github.com/rakutentech/go-alertnotification
func (w *WatchLogs) Notify(msg string) {
	err := errors.New(msg)
	ignoreErrs := []error{}
	alert := notifier.NewAlert(err, ignoreErrs)
	alert.Notify()
}

// SetInterval like JS
func (w *WatchLogs) SetInterval(action func(), interval time.Duration) func() {
	t := time.NewTicker(interval)
	q := make(chan struct{})
	go func() {
		for {
			select {
			case <-t.C:
				action()
			case <-q:
				t.Stop()
				return
			}
		}
	}()
	return func() {
		close(q)
	}
}
