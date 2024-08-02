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
  -log-level int
    	log level (0=info, 1=debug)
  -match string
    	regex for matching errors (empty to match all lines)
  -min int
    	on minimum num of matches, it should notify (default 1)
  -ms-teams-hook string
    	ms teams webhook
  -post string
    	run this shell command after every scan
  -proxy string
    	http proxy for webhooks
  -version
```


----

## Performance Notes

```sh
BenchmarkReadFileAndMatchErrors-10    	   10215	    122666 ns/op	    8684 B/op	      53 allocs/op
BenchmarkLoadAndSaveState-10          	   15241	     82295 ns/op	   10562 B/op	      38 allocs/op
BenchmarkLogRotation-10               	    9712	    143824 ns/op	    9707 B/op	      74 allocs/op
```


## CHANGE LOG

- v1.0.0: Initial release
- v2.0.0: Fixed log rotation issue and tail issues
- v2.0.11: Fixed invalid db issue on race, and use bytes instead of strings

