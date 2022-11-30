package server

import (
	"docker-repo/pkg/config"
	"docker-repo/pkg/log"
	"docker-repo/pkg/repo"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const LOG = "SERVER"

func Start() {
	// TODO
	// host := config.ServerName() + ":" + strconv.Itoa(config.Port())
	host := ":" + strconv.Itoa(config.Port())
	log.Info(LOG, "starting "+host)

	http.HandleFunc(config.ServerPath()+"/", route)
	http.HandleFunc(config.ServerTokenPath()+"/", routeToken)

	var err error
	if config.TlsCrtFile() != "" && config.TlsKeyFile() != "" {
		err = http.ListenAndServeTLS(host, config.TlsCrtFile(), config.TlsKeyFile(), nil)
	} else {
		err = http.ListenAndServe(host, nil)
	}
	if err != nil {
		log.Fatal(LOG, err.Error())
	}
}

func route(w http.ResponseWriter, r *http.Request) {
	if !checkHost(w, r) {
		return
	}
	switch r.Method {
	case "GET":
		handleGet(w, r)
	case "HEAD":
		handleHead(w, r)
	case "POST":
		handlePost(w, r)
	case "PATCH":
		handlePatch(w, r)
	case "PUT":
		handlePut(w, r)
	default:
		w.WriteHeader(404)
	}
}

func routeToken(w http.ResponseWriter, r *http.Request) {
	if !checkHost(w, r) {
		return
	}
	switch r.Method {
	case "GET":
		handleGetToken(w, r)
	default:
		w.WriteHeader(404)
	}
}

func handleGetToken(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	query.Get("account")
	query.Get("client_id")
	query.Get("offline_token")
	query.Get("service")

	auth := r.Header.Get("Authorization")

	if !strings.HasPrefix(auth, "Basic ") {
		w.WriteHeader(403)
		return
	}
	dec, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		log.Fatal(LOG, err.Error())
		w.WriteHeader(403)
		return
	}
	usrpwd := string(dec)
	sep := strings.Index(usrpwd, ":")
	if sep == -1 {
		w.WriteHeader(403)
		return
	}
	usr := usrpwd[:sep]
	pwd := usrpwd[sep+1:]

	if usr != config.Usr() || pwd != config.Pwd() {
		w.WriteHeader(403)
		return
	}

	rsp := map[string]any{
		"token": CreateToken(),
	}
	sendJson(w, 200, &rsp)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if !checkAuthToken(w, r) {
		return
	}
	w.WriteHeader(404)
}

func handleHead(w http.ResponseWriter, r *http.Request) {
	if !checkAuthToken(w, r) {
		return
	}

	// /v2/imagename/blobs/sha256:2279fc1f015f997d179c41693a6903e195f012a8dbe390d15bbc2f292b2da996
	paths := splitPath(r)
	if len(paths) != 4 || paths[2] != "blobs" {
		w.WriteHeader(404)
		return
	}

	img := paths[1]
	digest := paths[3]
	log.Info(LOG, "HEAD img "+img)

	exists, len, modified := repo.ExistsBlob(img, digest)

	if exists {
		setDefaultHeader(w)

		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
		w.Header().Set("Last-Modified", modified)
		w.Header().Set("Content-Length", strconv.FormatInt(len, 10))

		w.WriteHeader(200)
	} else {
		setDefaultHeader(w)
		w.WriteHeader(404)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	if !checkAuthToken(w, r) {
		return
	}

	// /v2/imagename/blobs/uploads
	paths := splitPath(r)
	if len(paths) != 4 || paths[2] != "blobs" || paths[3] != "uploads" {
		w.WriteHeader(404)
		return
	}

	img := paths[1]
	log.Info(LOG, "POST img "+img)

	setDefaultHeader(w)

	uploadUuid := repo.CreateBlobUploadUuid()
	uploadPath := repo.GetBlobUploadUrlPath(img, uploadUuid)
	w.Header().Set("Range", "0-0")
	w.Header().Set("Docker-Upload-UUID", uploadUuid)
	w.Header().Set("Location", uploadPath)
	w.Header().Set("Content-Length", "0")

	w.WriteHeader(202)
}

func handlePatch(w http.ResponseWriter, r *http.Request) {
	if !checkAuthToken(w, r) {
		return
	}

	// /v2/imagename/blobs/uploads/e81b3c5f-d53b-4ac1-b6e1-feaedc063cc7

	paths := splitPath(r)
	if len(paths) != 5 || paths[2] != "blobs" || paths[3] != "uploads" {
		w.WriteHeader(404)
		return
	}

	img := paths[1]
	uploadUuid := paths[4]
	uploadPath := repo.GetBlobUploadUrlPath(img, uploadUuid)
	log.Info(LOG, "PATCH img "+img+" uploadUuid "+uploadUuid)

	setDefaultHeader(w)

	len, err := repo.UploadBlob(img, uploadUuid, r.Body)

	if err != nil {
		log.Error(LOG, "upload blob failed: "+err.Error())
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Range", "0-"+strconv.FormatInt(len-1, 10))
	w.Header().Set("Docker-Upload-UUID", uploadUuid)
	w.Header().Set("Location", uploadPath)
	w.Header().Set("Content-Length", "0")

	w.WriteHeader(202)
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	if !checkAuthToken(w, r) {
		return
	}

	paths := splitPath(r)

	// /v2/imagename/blobs/uploads/uploadUid?digest=sha256%3A2279fc1f015f997d179c41693a6903e195f012a8dbe390d15bbc2f292b2da996
	if len(paths) == 5 && paths[2] == "blobs" && paths[3] == "uploads" {
		handlePutBlob(w, r)
		return
	}

	// /v2/imagename/manifests/version
	if len(paths) == 4 && paths[2] == "manifests" {
		handlePutManifest(w, r)
		return
	}
	w.WriteHeader(404)
}

func handlePutBlob(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	paths := splitPath(r)
	img := paths[1]
	uploadUuid := paths[4]
	digest := query.Get("digest")
	log.Info(LOG, "PUT (blob) img "+img+" uploadUuid "+uploadUuid+" digest "+digest)

	setDefaultHeader(w)

	len, uri, digest, err := repo.PutBlob(img, uploadUuid, digest)
	if err != nil {
		log.Error(LOG, "put blob failed: "+err.Error())
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Range", "0-"+strconv.FormatInt(len-1, 10))
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Location", uri)

	w.WriteHeader(201)
}

func handlePutManifest(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)
	img := paths[1]
	version := paths[3]

	log.Info(LOG, "PUT (manifest) img "+img+" version "+version)

	setDefaultHeader(w)

	digest, modified, content, err := repo.UploadManifest(img, version, r.Body)

	if err != nil {
		log.Error(LOG, "upload manifest failed: "+err.Error())
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Last-Modified", modified)
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))

	w.WriteHeader(201)

	_, err = w.Write(content)
	if err != nil {
		log.Error(LOG, "failed to write content: "+err.Error())
	}
}

func checkHost(w http.ResponseWriter, r *http.Request) bool {
	host := r.Host
	if strings.Contains(host, ":") {
		host = host[:strings.Index(host, ":")]
	}

	log.Info(LOG, "METHOD: "+r.Method+" HOST: "+host+" PATH: "+r.URL.Path)

	if host != config.ServerName() {
		w.WriteHeader(404)
		return false
	}
	return true
}

func checkAuthToken(w http.ResponseWriter, r *http.Request) bool {
	token := r.Header.Get("Authorization")

	if !ValidateToken(token) {
		tokenUrl := config.ServerAddress() + config.ServerTokenPath()
		setDefaultHeader(w)
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s",service="%s"`, tokenUrl, tokenUrl))

		sendError(w, 401, "UNAUTHORIZED", "access to the requested resource is not authorized")
		return false
	}

	// ----------------------------------------
	// REQ 1
	// ----------
	// GET /v2/token?account=admin&client_id=docker&offline_token=true&service=https%3A%2F%2Fnexton%2Fv2%2Ftoken HTTP/1.0
	// Host: nexton
	// X-Real-IP: 192.168.0.3
	// X-Forwarded-For: 192.168.0.3
	// X-Forwarded-Proto: https
	// Connection: close
	// User-Agent: docker/19.03.12 go/go1.13.10 git-commit/48a66213fe kernel/4.19.130-boot2docker os/linux arch/amd64 UpstreamClient(Docker-Client/19.03.
	// Authorization: Basic YWRtaW46bWlrZQ==
	// Accept-Encoding: gzip

	// no content

	// ----------------------------------------
	// RSP 1
	// ----------
	// HTTP/1.1 200 OK
	// Date: Tue, 29 Nov 2022 22:04:27 GMT
	// Server: Nexus/3.42.0-01 (OSS)
	// X-Content-Type-Options: nosniff
	// Content-Security-Policy: sandbox allow-forms allow-modals allow-popups allow-presentation allow-scripts allow-top-navigation
	// X-XSS-Protection: 1; mode=block
	// Content-Type: application/json
	// Content-Length: 60

	// {"token":"DockerToken.3585cda7-b86a-3398-8f3f-755f60b8af24"}
	//Authorization: Bearer DockerToken.3585cda7-b86a-3398-8f3f-755f60b8af24
	return true
}

func setDefaultHeader(w http.ResponseWriter) {
	w.Header().Set("Server", "mosi/0.1")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox allow-forms allow-modals allow-popups allow-presentation allow-scripts allow-top-navigation")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
}

func sendError(w http.ResponseWriter, status int, code, msg string) {
	var errors []any
	errors = append(errors, createError(code, msg))

	rsp := map[string]any{
		"errors": errors,
	}
	sendJson(w, status, &rsp)
}

func createError(code, msg string) any {
	err := map[string]any{
		"code":    code,
		"message": msg,
		"detail":  nil,
	}
	return err
}

func sendJson(w http.ResponseWriter, status int, rsp *map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(rsp)
}

func splitPath(r *http.Request) []string {
	paths := strings.Split(r.URL.Path, "/")
	var ret []string
	for _, path := range paths {
		if path != "" {
			ret = append(ret, path)
		}
	}
	return ret
}
