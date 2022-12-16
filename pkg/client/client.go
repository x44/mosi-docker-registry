package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mosi-docker-registry/pkg/app"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/filesys"
	"mosi-docker-registry/pkg/terminal"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func printCert(cert *x509.Certificate) {
	fmt.Printf("Certificate Issuer       : %v\n", cert.Issuer)
	fmt.Printf("Certificate Subject      : %v\n", cert.Subject)
	fmt.Printf("Certificate DNS Names    : %v\n", cert.DNSNames)
	fmt.Printf("Certificate IP Addresses : %v\n", cert.IPAddresses)
}

func getFn(host string, port int) string {
	return fmt.Sprintf("%s_%d", host, port)
}

func getCertDir() string {
	return filepath.Join(app.GetHomeDir(), ".mosi", "certs")
}

func getTokenDir() string {
	return filepath.Join(app.GetHomeDir(), ".mosi", "tokens")
}

func getCertPath(host string, port int) string {
	return filepath.Join(getCertDir(), getFn(host, port))
}

func getTokenPath(host string, port int) string {
	return filepath.Join(getTokenDir(), getFn(host, port))
}

func saveCert(cert *x509.Certificate, host string, port int) {
	_, err := filesys.WriteBytes(getCertPath(host, port), cert.Raw)
	app.CheckError("Failed to save certificate", err)
}

func isTrustedCert(cert *x509.Certificate, host string, port int) bool {
	pb, err := filesys.ReadBytes(getCertPath(host, port))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	app.CheckError("Failed to read certificate", err)
	a := cert.Raw
	b := *pb
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func saveToken(token, host string, port int) {
	_, err := filesys.WriteBytes(getTokenPath(host, port), []byte(token))
	app.CheckError("Failed to save token", err)
}

func loadToken(host string, port int) string {
	pb, err := filesys.ReadBytes(getTokenPath(host, port))
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	app.CheckError("Failed to read token", err)
	return string(*pb)
}

func createClient(args []string) *mosiClient {
	host := ""
	port := 443

	server := app.ArgsString("-s", "", &args)
	if len(server) > 0 {
		sa := strings.Split(server, ":")
		host = sa[0]
		if len(sa) > 1 {
			var err error
			port, err = strconv.Atoi(sa[1])
			app.CheckError("Invalid port in '"+server+"'", err)
			if port <= 0 || port > 65535 {
				app.CheckError("Invalid port in '"+server+"'", errors.New("port out of range"))
			}
		}
	} else {
		host = config.ServerOrProxyHost()
		port = config.ServerOrProxyPort()
	}

	usr := app.ArgsString("-u", "", &args)
	pwd := app.ArgsString("-p", "", &args)

	client := New(host, port, usr, pwd)
	return &client
}

type mosiClient struct {
	protocol  string
	host      string
	port      int
	usr       string
	pwd       string
	token     string
	client    *http.Client
	transport *http.Transport
}

func New(host string, port int, usr, pwd string) mosiClient {
	httpTransport := http.DefaultTransport.(*http.Transport)
	httpClient := &http.Client{Transport: httpTransport}

	return mosiClient{
		protocol:  "https",
		host:      host,
		port:      port,
		usr:       usr,
		pwd:       pwd,
		token:     loadToken(host, port),
		client:    httpClient,
		transport: httpTransport,
	}
}

func (c *mosiClient) makeUrl(urlOrPath string) string {
	if strings.HasPrefix(urlOrPath, "https://") || strings.HasPrefix(urlOrPath, "http://") {
		return urlOrPath
	}
	sep := ""
	if len(urlOrPath) > 0 {
		sep = "/"
	}
	return fmt.Sprintf("%s://%s:%d%s%s", c.protocol, c.host, c.port, sep, urlOrPath)
}

func (c *mosiClient) stringContent(rsp *http.Response) (string, error) {
	b, err := io.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *mosiClient) jsonContent(rsp *http.Response) (*map[string]interface{}, error) {
	m := map[string]any{}
	err := json.NewDecoder(rsp.Body).Decode(&m)
	return &m, err
}

func (c *mosiClient) shouldRetry(err error, cnt *int) bool {
	if err == nil || *cnt > 0 {
		return false
	}

	*cnt++

	urlErr := err.(*url.Error)
	if x509Err, ok := urlErr.Err.(x509.UnknownAuthorityError); ok {

		cert := x509Err.Cert

		if !isTrustedCert(cert, c.host, c.port) {

			fmt.Printf("The server's certificate is signed by an unknown authority.\n")
			printCert(cert)
			fmt.Printf("\nIf you trust this server, the certificate will be stored in %s\n", getCertDir())
			if !terminal.InputBool("Trust this server", "") {
				os.Exit(1)
			}

			saveCert(cert, c.host, c.port)
		}
		// cert already was or now is trusted
		sysCertPool, err := x509.SystemCertPool()
		app.CheckError("Failed to get system cert pool", err)
		sysCertPool.AddCert(x509Err.Cert)

		c.transport.TLSClientConfig = &tls.Config{RootCAs: sysCertPool}

		return true
	} else {
		app.CheckError("", err)
		// never reached:
		return false
	}
}

func (c *mosiClient) do(req *http.Request) *http.Response {
	// init auth with existing token (which may be "")
	c.setTokenAuth(req)

	var rsp *http.Response
	var err error
	var cnt = 0
	// server certificate check
	for {
		if rsp, err = c.client.Do(req); !c.shouldRetry(err, &cnt) {
			break
		}
	}
	// server certificate accepted

	if rsp.StatusCode == 401 {
		// auth with existing token failed
		c.inputUserAndPassword()
		if c.updateToken(rsp) {
			// got new token via basic auth
			c.setTokenAuth(req)
		} else {
			// use basic auth
			c.setBasicAuth(req)
		}
		rsp, err = c.client.Do(req)
		app.CheckError("", err)
	}
	return rsp
}

func (c *mosiClient) setTokenAuth(req *http.Request) {
	if len(c.token) > 0 {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *mosiClient) setBasicAuth(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.usr, c.pwd)))
	req.Header.Set("Authorization", "Basic "+auth)
}

func (c *mosiClient) inputUserAndPassword() {
	if len(c.usr) == 0 || len(c.pwd) == 0 {
		usr, pwd, err := terminal.InputUserAndPassword(c.usr)
		app.CheckError("Failed to read usr/pwd", err)
		c.usr = usr
		c.pwd = pwd
	}
}

func (c *mosiClient) getTokenAuthUrl(rsp *http.Response) string {
	authMethods := rsp.Header.Values("WWW-Authenticate")
	for _, authMethod := range authMethods {
		aml := strings.ToLower(authMethod)
		if strings.HasPrefix(aml, "bearer ") {
			a := strings.Split(authMethod[7:], ",")
			for _, e := range a {
				e := strings.TrimSpace(e)
				if strings.HasPrefix(e, "service=") {
					return e[9 : len(e)-1]
				}
			}
		}
	}
	return ""
}

// get, set and save token by sending basic auth to the token auth URL
func (c *mosiClient) updateToken(initialRsp *http.Response) bool {
	tokenAuthUrl := c.getTokenAuthUrl(initialRsp)
	if tokenAuthUrl == "" {
		return false
	}

	req := c.makeRequest("GET", tokenAuthUrl, nil)
	c.setBasicAuth(req)
	rsp, err := c.client.Do(req)

	if err != nil || rsp.StatusCode != 200 {
		return false
	}

	json, err := c.jsonContent(rsp)
	if err != nil {
		return false
	}
	if token, ok := (*json)["token"].(string); ok {
		c.token = token
		saveToken(token, c.host, c.port)
		return true
	}
	return false
}

func (c *mosiClient) makeRequest(method string, urlOrPath string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, c.makeUrl(urlOrPath), body)
	app.CheckError("Failed to create request", err)
	return req
}

func (c *mosiClient) Get(path string) *map[string]interface{} {
	req := c.makeRequest("GET", path, nil)
	rsp := c.do(req)

	if rsp.StatusCode != 200 {
		app.CheckError("", errors.New(rsp.Status))
	}

	result, err := c.jsonContent(rsp)
	app.CheckError("Failed to read JSON content", err)

	return result
}
