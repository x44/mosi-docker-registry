#!/bin/bash
MODULE=docker-repo
GOOS=windows GOARCH=amd64 go build -o bin/windows/$MODULE.exe cmd/$MODULE/*.go
GOOS=darwin GOARCH=amd64 go build -o bin/macos/$MODULE cmd/$MODULE/*.go
GOOS=linux GOARCH=amd64 go build -o bin/linux/$MODULE cmd/$MODULE/*.go
# MODULE=other
# GOOS=windows GOARCH=amd64 go build -o bin/windows/$MODULE.exe src/$MODULE/$MODULE.go
# GOOS=darwin GOARCH=amd64 go build -o bin/macos/$MODULE src/$MODULE/$MODULE.go
# GOOS=linux GOARCH=amd64 go build -o bin/linux/$MODULE src/$MODULE/$MODULE.go