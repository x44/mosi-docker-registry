#!/bin/bash
EXE=mosi
CMD=main
GOOS=windows GOARCH=amd64 go build -o bin/windows/$EXE.exe cmd/$CMD/*.go
GOOS=darwin GOARCH=amd64 go build -o bin/macos/$EXE cmd/$CMD/*.go
GOOS=linux GOARCH=amd64 go build -o bin/linux/$EXE cmd/$CMD/*.go
chmod +x bin/macos/$EXE
chmod +x bin/linux/$EXE
