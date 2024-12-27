<h1 align="center">
  Go Watch Logs
</h1>
<p align="center">
  Monitor static logs file for patterns and send alerts to MS Teams<br>
  Low Memory Footprint<br>
</p>

**Quick Setup:** One command to install.

**Hassle Free:** Doesn't require root or sudo.

**Platform:** Supports (arm64, arch64, Mac, Mac M1, Ubuntu and Windows).

**Flexible:** Works with any logs file, huge to massive, log rotation is supported.

**Notify:** Supports MS Teams or emails.

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

# match 50 and 40 errors on ltsv log
go-watch-logs --file-path=my.log --match='HTTP/1.1" 50|HTTP/1.1" 40'

# match 50x and 40x errors on ltsv log, and ignore 404
go-watch-logs --file-path=my.log --match='HTTP/1.1" 50|HTTP/1.1" 40' --ignore='HTTP/1.1" 404'

# match 50x and run every 60 seconds
go-watch-logs --file-path=my.log --match='HTTP/1.1" 50' --every=60
```

### Watching a log file for anomalies

```sh
```


**All done!**

## Help

```sh
  -db-path string
    	path to store db file (default "/Users/pulkit.kathuria/.go-watch-logs.db")
  -every uint
    	run every n seconds (0 to run once)
  -f string
    	(short for --file-path) full path to the log file
  -file-path string
    	full path to the log file
  -file-paths-cap int
    	max number of file paths to watch (default 100)
  -health-check-every uint
    	run health check every n seconds (0 to disable)
  -ignore string
    	regex for ignoring errors (empty to ignore none)
  -log-level int
    	log level (0=info, -4=debug, 4=warn, 8=error)
  -match string
    	regex for matching errors (empty to match all lines)
  -mem-limit int
    	memory limit in MB (0 to disable) (default 100)
  -min int
    	on minimum num of matches, it should notify (default 1)
  -ms-teams-hook string
    	ms teams webhook
  -post-always string
    	run this shell command after every scan
  -post-min string
    	run this shell command after every scan when min errors are found
  -proxy string
    	http proxy for webhooks
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
BenchmarkReadFileAndMatchErrors-10    	     969	   1173870 ns/op	   12920 B/op	     146 allocs/op
BenchmarkLoadAndSaveState-10          	    5296	    230536 ns/op	    9179 B/op	     180 allocs/op
BenchmarkLogRotation-10               	    1036	   1175464 ns/op	   12930 B/op	     146 allocs/op
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

