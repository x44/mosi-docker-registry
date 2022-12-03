package main

import (
	"mosi-docker-repo/pkg/config"
	"mosi-docker-repo/pkg/server"
)

func main() {
	config.ReadConfig()
	server.Start()
}
