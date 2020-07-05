#! /bin/sh
GOOS=linux GOARCH=amd64 go build cmd/watch_logs.go; mv watch_logs ./bin/linux/