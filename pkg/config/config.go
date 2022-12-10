package config

import (
	"encoding/json"
	"io"
	"mosi-docker-repo/pkg/logging"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const LOG = "Config"

type config struct {
	Repo     repo      `json:"repo"`
	Server   server    `json:"server"`
	Proxy    proxy     `json:"proxy"`
	Log      log       `json:"log"`
	Accounts []account `json:"accounts"`
}

type repo struct {
	Dir                string `json:"dir"`
	AllowAnonymousPull bool   `json:"allowAnonymousPull"`
}

type server struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	TlsCrtFile string `json:"tlsCrtFile"`
	TlsKeyFile string `json:"tlsKeyFile"`
}

type proxy struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type log struct {
	ServiceLevel string `json:"serviceLevel"`
	ConsoleLevel string `json:"consoleLevel"`
	LogFileLevel string `json:"logFileLevel"`
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

var cwd string
var cfg config

func makeAbs(fn string) string {
	if filepath.IsAbs(fn) {
		return fn
	}
	return filepath.Join(cwd, fn)
}

func RepoDir() string {
	return makeAbs(cfg.Repo.Dir)
}

func AllowAnonymousPull() bool {
	return cfg.Repo.AllowAnonymousPull
}

func ServerHost() string {
	return cfg.Server.Host
}

func ServerPort() int {
	return cfg.Server.Port
}

func ServerAddress() string {
	return ServerHost() + ":" + strconv.Itoa(ServerPort())
}

func TlsCrtFile() string {
	if cfg.Server.TlsCrtFile == "" {
		return cfg.Server.TlsCrtFile
	}
	return makeAbs(cfg.Server.TlsCrtFile)
}

func TlsKeyFile() string {
	if cfg.Server.TlsCrtFile == "" {
		return cfg.Server.TlsKeyFile
	}
	return makeAbs(cfg.Server.TlsKeyFile)
}

func TlsEnabled() bool {
	return TlsCrtFile() != "" && TlsKeyFile() != ""
}

// Returns the server's address as it can be called by clients.
// Takes into account an upstream reverse proxy.
func ServerUrl(r *http.Request) string {

	host := strings.Split(r.Host, ":")[0]
	port := ServerPort()

	// Note that it actually makes to sense to use http here,
	// since all requests to either Mosi or to the reverse proxy MUST be https.
	// However, for testing purposes, we also use http.
	isTls := TlsEnabled()
	protocol := "http"
	if isTls {
		protocol = "https"
	}

	// overwrite with request proxy header fields
	reqPort := r.Header.Get("X-Forwarded-Port")
	if reqPort != "" {
		p, err := strconv.Atoi(reqPort)
		if err == nil {
			port = p
		}
		protocol = "https"
	}

	// overwrite with config proxy settings
	if cfg.Proxy.Host != "" {
		host = cfg.Proxy.Host
		protocol = "https"
	}
	if cfg.Proxy.Port > 0 {
		port = cfg.Proxy.Port
		protocol = "https"
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

func LogLevelService() int {
	return logging.Level(cfg.Log.ServiceLevel)
}

func LogLevelConsole() int {
	return logging.Level(cfg.Log.ConsoleLevel)
}

func LogLevelFile() int {
	return logging.Level(cfg.Log.LogFileLevel)
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
	cfg.Repo = repo{
		Dir:                "repo",
		AllowAnonymousPull: true,
	}

	cfg.Server = server{
		Host:       "mosi",
		Port:       4444,
		TlsCrtFile: "certs/mosi-example.crt",
		TlsKeyFile: "certs/mosi-example.key",
	}

	cfg.Proxy = proxy{
		Host: "",
		Port: 0,
	}

	cfg.Log = log{
		ServiceLevel: "INFO",
		ConsoleLevel: "INFO",
		LogFileLevel: "INFO",
	}

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

func writeConfig(fn string) {
	dir := filepath.Dir(fn)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		logging.Error(LOG, "failed to create %s", dir)
		return
	}
	f, err := os.Create(fn)
	if err != nil {
		logging.Error(LOG, "failed to create %s", fn)
		return
	}
	defer f.Close()

	buf, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		logging.Error(LOG, "failed to marshal %s", fn)
		return
	}

	_, err = f.Write(buf)
	if err != nil {
		logging.Error(LOG, "failed to write %s", fn)
		return
	}
}

func ReadConfig(workdir, fn string) {
	cwd = workdir
	initDefaults()
	f, err := os.Open(fn)
	if err != nil {
		if os.IsNotExist(err) {
			logging.Info(LOG, "file not found, creating default %s", fn)
		} else {
			logging.Fatal(LOG, "%s %v", fn, err)
		}
	} else {
		buf, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			logging.Error(LOG, "failed to read, creating default %s", fn)
		}

		err = json.Unmarshal(buf, &cfg)
		if err != nil {
			logging.Error(LOG, "failed to unmarshal %s", fn)
		}
	}

	writeConfig(fn)
}
