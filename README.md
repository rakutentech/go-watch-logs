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
    	ignore pattern
  -match string
    	match pattern
  -ms-teams-hook string
    	ms teams webhook
  -version
    	print version
```


----

## Performance Notes

```
```


## Credits

1. https://github.com/rakutentech/go-alertnotification

