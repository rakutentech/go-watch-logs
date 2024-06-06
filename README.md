<h1 align="center">
  Go Watch Logs
</h1>
<p align="center">
  Monitor static logs file for patterns and send alerts to MS Teams<br>
  Zero memory allocation<br>
</p>

**Quick Setup:** One command to install Go and manage versions.

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


**All done!**

## Help

```sh
  -db-path string
    	path to store db file (default ".go-watch-logs.db")
  -every uint
    	run every n seconds (0 to run once)
  -file-path string
    	full path to the log file
  -ignore string
    	regex for ignoring errors (empty to ignore none)
  -match string
    	regex for matching errors (empty to match all lines)
  -min-error int
    	on minimum num of errors should notify (default 1)
  -ms-teams-hook string
    	ms teams webhook
  -no-cache
    	read back from the start of the file (default false)
  -proxy string
    	http proxy for webhooks
  -version
```


----

## Performance Notes

```sh
BenchmarkReadFileAndMatchErrors-10    	   10816	    112233 ns/op	    8591 B/op	      50 allocs/op
BenchmarkSetAndGetLastLineNum-10      	 3231630	       371.3 ns/op	     490 B/op	       8 allocs/op
BenchmarkLoadAndSaveState-10          	   15504	     78520 ns/op	   11563 B/op	     104 allocs/op
BenchmarkLogRotation-10               	    9450	    122679 ns/op	    9445 B/op	      69 allocs/op
```


## Credits

1. https://github.com/rakutentech/go-alertnotification

