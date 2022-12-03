package server

import (
	"encoding/json"
	"fmt"
	"mosi-docker-repo/pkg/config"
	"mosi-docker-repo/pkg/log"
	"mosi-docker-repo/pkg/repo"
	"net/http"
	"strconv"
	"strings"
)

const LOG = "SERVER"

func Start() {
	addr := config.ServerAddress()
	log.Info(LOG, "starting "+addr)

	http.HandleFunc(config.ServerPath()+"/", route)       // trailing / is required
	http.HandleFunc(config.ServerTokenPath(), routeToken) // trailing / not allowed, otherwise all /v2/token?xxx requests get redirected

	var err error
	if config.TlsEnabled() {
		err = http.ListenAndServeTLS(addr, config.TlsCrtFile(), config.TlsKeyFile(), nil)
	} else {
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		log.Fatal(LOG, err.Error())
	}
}

func route(w http.ResponseWriter, r *http.Request) {
	PrintReq(r)
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
	PrintReq(r)
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
	token := CreateAndStoreTokenFromBasicAuth(r)
	if token == "" {
		w.WriteHeader(403)
		return
	}

	setDefaultHeader(w)

	rsp := map[string]any{
		"token": token,
	}
	sendJson(w, 200, &rsp)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)

	// /v2/
	if len(paths) == 1 {
		if !checkRootAuth(w, r) {
			return
		}
		w.WriteHeader(200)
		return
	}

	// /v2/imagename/blobs/digest
	if len(paths) == 4 && paths[2] == "blobs" {
		handleGetBlob(w, r)
		return
	}

	// /v2/imagename/manifests/digest
	if len(paths) == 4 && paths[2] == "manifests" {
		handleGetManifest(w, r)
		return
	}
	w.WriteHeader(404)
}

func handleGetBlob(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)
	img := paths[1]
	digest := paths[3]

	if !checkPullAuth(w, r, img) {
		return
	}

	setDefaultHeader(w)

	repo.DownloadBlob(img, digest, w)
}

func handleGetManifest(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)
	img := paths[1]
	digest := paths[3]

	if !checkPullAuth(w, r, img) {
		return
	}

	setDefaultHeader(w)

	repo.DownloadManifest(img, digest, w)
}

func handleHead(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)
	img := paths[1]

	if !checkPullAuth(w, r, img) {
		return
	}

	// /v2/imagename/blobs/digest
	if len(paths) == 4 && paths[2] == "blobs" {
		handleHeadBlob(w, r)
		return
	}

	// /v2/imagename/manifests/latest
	if len(paths) == 4 && paths[2] == "manifests" {
		handleHeadManifest(w, r)
		return
	}
	w.WriteHeader(404)
}

func handleHeadBlob(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)
	img := paths[1]
	digest := paths[3]

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

func handleHeadManifest(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)
	img := paths[1]
	tag := paths[3]

	exists, len, modified, digest := repo.ExistsManifest(img, tag)

	if exists {
		setDefaultHeader(w)

		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.Header().Set("Last-Modified", modified)
		w.Header().Set("Content-Length", strconv.FormatInt(len, 10))

		w.WriteHeader(200)
	} else {
		setDefaultHeader(w)
		w.WriteHeader(404)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	// /v2/imagename/blobs/uploads
	paths := splitPath(r)
	if len(paths) != 4 || paths[2] != "blobs" || paths[3] != "uploads" {
		w.WriteHeader(404)
		return
	}

	img := paths[1]

	if !checkPushAuth(w, r, img) {
		return
	}

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
	// /v2/imagename/blobs/uploads/uploadUuid
	paths := splitPath(r)
	if len(paths) != 5 || paths[2] != "blobs" || paths[3] != "uploads" {
		w.WriteHeader(404)
		return
	}

	img := paths[1]
	uploadUuid := paths[4]

	if !checkPushAuth(w, r, img) {
		return
	}

	uploadPath := repo.GetBlobUploadUrlPath(img, uploadUuid)

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
	paths := splitPath(r)
	img := paths[1]

	if !checkPushAuth(w, r, img) {
		return
	}

	// /v2/imagename/blobs/uploads/uploadUid?digest=sha256%3A2279fc1f015f997d179c41693a6903e195f012a8dbe390d15bbc2f292b2da996
	if len(paths) == 5 && paths[2] == "blobs" && paths[3] == "uploads" {
		handlePutBlob(w, r)
		return
	}

	// /v2/imagename/manifests/tag
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

	setDefaultHeader(w)

	len, uri, digest, err := repo.PutBlob(img, uploadUuid, digest, r)
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
	tag := paths[3]

	setDefaultHeader(w)

	digest, modified, content, err := repo.UploadManifest(img, tag, r.Body)

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

	if host != config.ServerHost() {
		w.WriteHeader(404)
		return false
	}
	return true
}

func checkRootAuth(w http.ResponseWriter, r *http.Request) bool {
	return checkAuth(w, r, "", false, false)
}

func checkPullAuth(w http.ResponseWriter, r *http.Request, img string) bool {
	return checkAuth(w, r, img, true, false)
}

func checkPushAuth(w http.ResponseWriter, r *http.Request, img string) bool {
	return checkAuth(w, r, img, true, true)
}

func checkAuth(w http.ResponseWriter, r *http.Request, img string, allowAnonymous, wantPush bool) bool {
	if CheckAuth(r, img, allowAnonymous, wantPush) {
		return true
	}

	setDefaultHeader(w)

	tokenUrl := config.ServerUrl(r) + config.ServerTokenPath()
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s", service="%s"`, tokenUrl, tokenUrl))
	if allowAnonymous {
		w.Header().Add("WWW-Authenticate", fmt.Sprintf(`BASIC realm="%s"`, "Mosi Docker Repository"))
	}

	sendError(w, 401, "UNAUTHORIZED", "access to the requested resource is not authorized")
	return false
}

func setDefaultHeader(w http.ResponseWriter) {
	w.Header().Set("Server", "Mosi Docker Repository/0.1")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox allow-forms allow-modals allow-popups allow-presentation allow-scripts allow-top-navigation")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Docker-Distribution-Api-Tag", "registry/2.0")
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
