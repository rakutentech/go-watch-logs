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
  -version
```


----

## Performance Notes

```
BenchmarkReadFileAndMatchErrors-10    	   10328	    113576 ns/op	    8543 B/op	      49 allocs/op
BenchmarkReadFileAndMatchErrors-10    	   10000	    115039 ns/op	    8543 B/op	      49 allocs/op
BenchmarkSetAndGetLastLineNum-10      	 3247788	       369.2 ns/op	     490 B/op	       8 allocs/op
BenchmarkSetAndGetLastLineNum-10      	 3224370	       370.3 ns/op	     490 B/op	       8 allocs/op
BenchmarkLoadAndSaveState-10          	   14349	     74996 ns/op	   11479 B/op	     103 allocs/op
BenchmarkLoadAndSaveState-10          	   15531	     79613 ns/op	   12916 B/op	     177 allocs/op
BenchmarkLogRotation-10               	    9644	    128297 ns/op	   10326 B/op	      89 allocs/op
BenchmarkLogRotation-10               	    9604	    126398 ns/op	   10295 B/op	      88 allocs/op
```


## Credits

1. https://github.com/rakutentech/go-alertnotification

