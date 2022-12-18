package server

import (
	"log"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/json"
	"mosi-docker-registry/pkg/logging"
	"mosi-docker-registry/pkg/repo"
	"net/http"
	"strconv"
	"strings"
)

const LOG = "SERVER"

type serverErrorWriter struct {
}

func (w *serverErrorWriter) Write(buf []byte) (int, error) {
	n := len(buf)
	// Do NOT log the following errors:
	// 1) http: TLS handshake error from 1.2.3.4:12345: remote error: tls: bad certificate
	s := string(buf[:n-1])
	if !strings.HasSuffix(s, "tls: bad certificate") {
		logging.Error(LOG, s)
	}
	return n, nil
}

func Start(version string) {
	addr := config.ServerAddress()
	protocol := "http"
	if config.TlsEnabled() {
		protocol = "https"
	}
	logging.Info(LOG, "Mosi %s address %s://%s, repository %s", version, protocol, addr, config.RepoDir())

	http.HandleFunc(config.ServerPath()+"/", route)       // trailing / is required
	http.HandleFunc(config.ServerTokenPath(), routeToken) // trailing / not allowed, otherwise all /v2/token?xxx requests get redirected

	serverErrorWriter := &serverErrorWriter{}
	serverErrorLogger := log.New(serverErrorWriter, "", 0)

	srv := &http.Server{
		Addr:     addr,
		ErrorLog: serverErrorLogger,
	}
	var err error
	if config.TlsEnabled() {
		err = srv.ListenAndServeTLS(config.TlsCrtFile(), config.TlsKeyFile())
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		logging.Fatal(LOG, "%s", err.Error())
	}
}

func route(w http.ResponseWriter, r *http.Request) {
	printRequest(r)
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
	printRequest(r)
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
	token := createAndStoreTokenFromBasicAuth(r)
	if token == "" {
		w.WriteHeader(403)
		return
	}

	setDefaultHeader(w)

	rsp := json.NewJsonObject()
	rsp.Put("token", token)
	sendJson(w, 200, rsp)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	paths := splitPath(r)

	// /v2/
	if len(paths) == 1 {
		if !initAuth(w, r) {
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

	// /v2/cli/...
	if len(paths) > 1 && paths[1] == "cli" {
		cliHandleGet(w, r)
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
		logging.Error(LOG, "upload blob failed: %s", err.Error())
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
		logging.Error(LOG, "put blob failed: %s", err.Error())
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
		logging.Error(LOG, "upload manifest failed: %s", err.Error())
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
		logging.Error(LOG, "failed to write content: %s", err.Error())
	}
}
