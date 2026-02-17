<h1 align="center">
  Go Watch Logs
</h1>
<p align="center">
  Monitor static logs file for patterns and send alerts to MS Teams & PagerDuty<br>
  Low Memory Footprint<br>
</p>

**Quick Setup:** One command to install.

**Hassle Free:** Doesn't require root or sudo.

**Platform:** Supports (arm64, arch64, Mac, Mac M1, Ubuntu and Windows).

**Flexible:** Works with any logs file, huge to massive, log rotation is supported.

**Notify:** Supports MS Teams, PagerDuty.

**Scheduler:** Run it on a cron.

### Install using go

```bash
go install github.com/rakutentech/go-watch-logs@latest
go-watch-logs --help
```

### Install using curl

Use this method if go is not installed on your server

```bash
curl -sL https://raw.githubusercontent.com/rakutentech/go-watch-logs/master/install.sh | sh
./go-watch-logs --help
```

## Examples

### Watching a log file for errors

```sh
# match error patterns and notify on MS Teams
go-watch-logs --file-path=my.log --match="error:pattern1|error:pattern2" --ms-teams-hook="https://outlook.office.com/webhook/xxxxx"

# match error patterns and notify on PagerDuty
go-watch-logs --file-path=my.log --match="error:pattern1|error:pattern2" --pagerduty-key="YOUR_ROUTING_KEY" --pagerduty-dedupkey="uniq-name"

# notify both MS Teams and PagerDuty
go-watch-logs --file-path=my.log --match="error" --ms-teams-hook="https://..." --pagerduty-key="YOUR_ROUTING_KEY" --pagerduty-dedupkey="uniq-name"

# match 50 and 40 errors on ltsv log
go-watch-logs --file-path=my.log --match='HTTP/1.1" 50|HTTP/1.1" 40'

# match 50x and 40x errors on ltsv log, and ignore 404
go-watch-logs --file-path=my.log --match='HTTP/1.1" 50|HTTP/1.1" 40' --ignore='HTTP/1.1" 404'

# match 50x and run every 60 seconds
go-watch-logs --file-path=my.log --match='HTTP/1.1" 50' --every=60
```

**All done!**

## Help

```sh
  -every uint
    	run every n seconds (0 to run once)
  -f string
    	(short for --file-path) full path to the file to watch
  -file-path string
    	full path to the file to watch
  -file-paths-cap int
    	max number of file paths to watch (default 100)
  -file-recent-secs uint
    	only files modified in the last n seconds, 0 to disable (default 86400)
  -ignore string
    	regex for ignoring errors (empty to ignore none)
  -log-file string
    	full path to output log file. Empty will log to stdout
  -log-level int
    	log level (0=info, -4=debug, 4=warn, 8=error)
  -match string
    	regex for matching errors (empty to match all lines)
  -mbf int
    	max buffer in MB, default is 0 (not provided) for go's default 64KB
  -mem-limit int
    	memory limit in MB (0 to disable) (default 128)
  -min int
    	on minimum num of matches, it should notify (default 1)
  -ms-teams-hook string
    	ms teams webhook
  -pagerduty-key string
    	pagerduty routing/integration key
  -pagerduty-dedupkey string
    	pagerduty deduplication key
  -post-cmd string
    	run this shell command after every scan when min errors are found
  -proxy string
    	http proxy for webhooks
  -streak int
    	on minimum num of streak matches, it should notify (default 1)
  -test
    	Quickly test paths or regex
    	# will test if the input matches the regex
    	echo test123 | go-watch-logs --match=123 --test
    	# will test if the file paths are found and list them
    	go-watch-logs --file-path=./ssl_access.*log --test

  -version
```


----

## Performance Notes

```sh
$ go test -bench=. ./... -benchmem
BenchmarkReadFileAndMatchErrors-10    	   13588	     91900 ns/op	    8243 B/op	      43 allocs/op
BenchmarkLoadAndSaveState-10          	 3135621	     375.3 ns/op	     352 B/op	       8 allocs/op
BenchmarkLogRotation-10               	   13807	    101088 ns/op	    8243 B/op	      43 allocs/op
```

## Development Notes

```sh
go run main.go -file-path="testdata/*.log" --every=3
go test ./...
```


## CHANGE LOG

- **v1.0.0** Initial release
- **v1.0.1** Health Check
- **v1.0.2** `--post`, `--post-min`, `--health-check-every` and preview added
- **v1.0.7** Added percentage
- **v1.0.10** Better tint and added `-test` flag and `-f` for short form of `--file-path`
- **v1.0.12** Stable
- **v1.0.13** Performance improvements via singletons
- **v1.0.19** Global slog handler and notifier on own alerts
- **v1.1.0** Uses in memory state for faster performance, streaks added
- **v1.1.3** Stable Version
- **v1.1.4** Supports Geo Tagging via IP Address offline
- **v1.1.5** Supports Pager Duty
- **v1.1.6** Supports longer regex patterns by splitting them
- **v1.1.11** vup go 1.26

