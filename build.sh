#!/bin/bash
EXE=mosi
MODULE=docker-repo
GOOS=windows GOARCH=amd64 go build -o bin/windows/$EXE.exe cmd/$MODULE/*.go
GOOS=darwin GOARCH=amd64 go build -o bin/macos/$EXE cmd/$MODULE/*.go
GOOS=linux GOARCH=amd64 go build -o bin/linux/$EXE cmd/$MODULE/*.go
chmod +x bin/macos/$EXE
chmod +x bin/linux/$EXE
