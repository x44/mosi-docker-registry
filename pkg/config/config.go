package config

import (
	"docker-repo/pkg/log"
	"encoding/json"
	"io"
	"os"
	"strconv"
)

const LOG = "Config"

type config struct {
	Port       int    `json:"port"`
	ServerName string `json:"serverName"`
	TlsCrtFile string `json:"tlsCrtFile"`
	TlsKeyFile string `json:"tlsKeyFile"`
	Usr        string `json:"usr"`
	Pwd        string `json:"pwd"`
	// TODO remove
	DummyPort int    `json:"dummyPort"`
	RepoDir   string `json:"repoDir"`
}

var cfg config

func Port() int {
	return cfg.Port
}

func ServerName() string {
	return cfg.ServerName
}

func TlsCrtFile() string {
	return cfg.TlsCrtFile
}

func TlsKeyFile() string {
	return cfg.TlsKeyFile
}

func Usr() string {
	return cfg.Usr
}

func Pwd() string {
	return cfg.Pwd
}

func ServerAddress() string {
	var portStr = ""
	port := Port()
	if cfg.DummyPort > 0 {
		port = cfg.DummyPort
	}
	if port != 443 {
		portStr = ":" + strconv.Itoa(port)
	}
	return "https://" + ServerName() + portStr
}

func ServerPath() string {
	return "/v2"
}

func ServerTokenPath() string {
	return ServerPath() + "/token"
}

func RepoDir() string {
	return cfg.RepoDir
}

func initDefaults() {
	cfg.Port = 4444
	cfg.ServerName = "nexton"
	cfg.TlsCrtFile = "certs/nexton.crt"
	cfg.TlsKeyFile = "certs/nexton.key"
	cfg.Usr = "admin"
	cfg.Pwd = "mike"
	cfg.DummyPort = 0
	cfg.RepoDir = "repo"
}

func writeConfig() {
	f, err := os.Create("config.json")
	if err != nil {
		log.Fatal(LOG, "failed to create config.json")
		return
	}
	defer f.Close()

	buf, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		log.Fatal(LOG, "failed to marshal config")
		return
	}

	_, err = f.Write(buf)
	if err != nil {
		log.Fatal(LOG, "failed to write config.json")
		return
	}
}

func ReadConfig() {
	initDefaults()
	f, err := os.Open("config.json")
	if err != nil {
		log.Info(LOG, "config.json not found, creating default")
		return
	}
	buf, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		log.Info(LOG, "failed to read config.json")
	}

	err = json.Unmarshal(buf, &cfg)
	if err != nil {
		log.Info(LOG, "failed to unmarshal config")
	}

	writeConfig()
}
