package config

import (
	"encoding/json"
	"io"
	"mosi-docker-repo/pkg/log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const LOG = "Config"

type config struct {
	RepoDir            string    `json:"repoDir"`
	ServerHost         string    `json:"serverHost"`
	ServerPort         int       `json:"serverPort"`
	TlsCrtFile         string    `json:"tlsCrtFile"`
	TlsKeyFile         string    `json:"tlsKeyFile"`
	ProxyHost          string    `json:"proxyHost"`
	ProxyPort          int       `json:"proxyPort"`
	ProxyProtocol      string    `json:"proxyProtocol"`
	Accounts           []account `json:"accounts"`
	AllowAnonymousPull bool      `json:"allowAnonymousPull"`
}

type account struct {
	Usr    string  `json:"usr"`
	Pwd    string  `json:"pwd"`
	Images []image `json:"images"`
}

type image struct {
	Name string `json:"name"`
	Pull bool   `json:"pull"`
	Push bool   `json:"push"`
}

var cfg config

func RepoDir() string {
	return cfg.RepoDir
}

func ServerHost() string {
	return cfg.ServerHost
}

func ServerPort() int {
	return cfg.ServerPort
}

func ServerAddress() string {
	return ServerHost() + ":" + strconv.Itoa(ServerPort())
}

func TlsCrtFile() string {
	return cfg.TlsCrtFile
}

func TlsKeyFile() string {
	return cfg.TlsKeyFile
}

func TlsEnabled() bool {
	return TlsCrtFile() != "" && TlsKeyFile() != ""
}

// Returns the server's address, taking into account an upstream reverse proxy.
func ServerUrl(r *http.Request) string {

	host := strings.Split(r.Host, ":")[0]
	port := ServerPort()

	isTls := TlsEnabled()
	protocol := "http"
	if isTls {
		protocol = "https"
	}

	// override with request proxy header fields
	reqPort := r.Header.Get("X-Forwarded-Port")
	reqProtocol := r.Header.Get("X-Forwarded-Proto")
	if reqPort != "" {
		p, err := strconv.Atoi(reqPort)
		if err == nil {
			port = p
		}
	}
	if reqProtocol != "" {
		protocol = reqProtocol
	}

	// override with config proxy settings
	if cfg.ProxyHost != "" {
		host = cfg.ProxyHost
	}
	if cfg.ProxyPort > 0 {
		port = cfg.ProxyPort
	}
	if cfg.ProxyProtocol != "" {
		protocol = cfg.ProxyProtocol
	}

	var portStr = ":" + strconv.Itoa(port)
	if (port == 80 && !isTls) || (port == 443 && isTls) {
		portStr = ""
	}

	return protocol + "://" + host + portStr
}

func ServerPath() string {
	return "/v2"
}

func ServerTokenPath() string {
	return ServerPath() + "/token"
}

func AllowAnonymousPull() bool {
	return cfg.AllowAnonymousPull
}

func GetAccountImageAccessRights(usr, pwd string, allowAnonymous bool) (imagesAllowedToPull []string, imagesAllowedToPush []string) {
	imagesAllowedToPull = nil
	imagesAllowedToPush = nil

	usr, allowed := mapAndCheckAnonymousAccess(usr, allowAnonymous)
	if !allowed {
		return
	}

	// TODO de-uglify this
	for _, account := range cfg.Accounts {
		if account.Usr == usr {
			if account.Pwd == pwd {
				for _, image := range account.Images {
					if image.Pull {
						imagesAllowedToPull = append(imagesAllowedToPull, image.Name)
					}
					if image.Push {
						imagesAllowedToPush = append(imagesAllowedToPush, image.Name)
					}
				}
				return
			}
			return
		}
	}
	return
}

func GetScopeImageAccessRights(imageName, usr, pwd string, allowAnonymous bool) (imagesAllowedToPull []string, imagesAllowedToPush []string) {
	imagesAllowedToPull = nil
	imagesAllowedToPush = nil

	usr, allowed := mapAndCheckAnonymousAccess(usr, allowAnonymous)
	if !allowed {
		return
	}

	// TODO de-uglify this
	for _, account := range cfg.Accounts {
		if account.Usr == usr {
			if account.Pwd == pwd {
				for _, image := range account.Images {
					if image.Name == imageName || image.Name == "*" {
						if image.Pull {
							imagesAllowedToPull = append(imagesAllowedToPull, imageName)
						}
						if image.Push {
							imagesAllowedToPush = append(imagesAllowedToPush, imageName)
						}
						return
					}
				}
				return
			}
			return
		}
	}
	return
}

func mapAndCheckAnonymousAccess(usr string, allowAnonymous bool) (string, bool) {
	if usr == "" {
		usr = "anonymous"
	}

	if usr == "anonymous" && (!allowAnonymous || !AllowAnonymousPull()) {
		return usr, false
	}
	return usr, true
}

func initDefaults() {
	cfg.RepoDir = "repo"
	cfg.ServerHost = "nexton"
	cfg.ServerPort = 4444
	cfg.TlsCrtFile = "certs/nexton.crt"
	cfg.TlsKeyFile = "certs/nexton.key"
	cfg.ProxyHost = ""
	cfg.ProxyPort = 0
	cfg.ProxyProtocol = ""
	cfg.AllowAnonymousPull = true

	cfg.Accounts = []account{
		{
			Usr: "admin",
			Pwd: "admin",
			Images: []image{
				{
					Name: "*",
					Pull: true,
					Push: true,
				},
			},
		},
		{
			Usr: "anonymous",
			Pwd: "",
			Images: []image{
				{
					Name: "*",
					Pull: true,
					Push: false,
				},
			},
		},
	}
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
