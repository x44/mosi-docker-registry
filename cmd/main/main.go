package main

import (
	"docker-repo/pkg/config"
	"docker-repo/pkg/server"
)

func main() {
	config.ReadConfig()
	server.Start()
}
