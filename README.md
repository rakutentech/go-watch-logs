<p align="center">
  Go Watch Logs<br>
  Monitor static logs file for patterns and send alerts to MS Teams<br>
</p>

**Quick Setup:** One command to install Go and manage versions.

**Hassle Free:** Doesn't require root or sudo.

**Platform:** Supports (arm64, arch64, Mac, Mac M1, Ubuntu and Windows).

**Flexible:** Works with any logs file, huge to massive, log rotation is supported.

**Notify:** Supports MS Teams. Emails, Slack (coming soon).

### Install using go

```bash
go install github.com/rakutentech/go-watch-logs@latest
go-watch-logs --help
```

### Install using curl

Use this method if go is not installed on your server

```bash
curl -sL https://raw.githubusercontent.com/rakutentech/go-watch-logs/master/install.sh | sh
```

## Run it on a cron


```
* * * * * go-watch-logs --file-path=my.log --match="error:pattern1|error:pattern2" --ms-teams-hook="https://outlook.office.com/webhook/xxxxx"
```


**All done!**

## Help

```sh
  -db-path string
    	path to db file (default "go-watch-logs.db")
  -file-path string
    	path to logs file
  -ignore string
    	regex for ignoring errors
  -match string
    	regex for matching errors
  -min-error int
    	on minimum error threshold to notify (default 1)
  -ms-teams-hook string
    	ms teams webhook
  -version
    	print version
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

