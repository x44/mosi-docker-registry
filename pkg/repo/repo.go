package repo

import (
	"docker-repo/pkg/config"
	"docker-repo/pkg/filesys"
	"docker-repo/pkg/log"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const LOG = "REPO"

func ExistsBlob(img, digest string) (exists bool, len int64, modified string) {
	exists = false
	len = -1
	modified = ""

	servedFn, err := getBlobServedFilename(img, digest)
	if err != nil {
		return
	}

	len, err = filesys.Size(servedFn)
	if err != nil {
		return
	}

	modified, err = filesys.ModifiedHttpDate(servedFn)
	if err != nil {
		return
	}

	exists = true
	return
}

func CreateBlobUploadUuid() string {
	uploadUuid := uuid.New().String()
	return uploadUuid
}

// /v2/imagename/blobs/upload/uploadUuid
func GetBlobUploadUrlPath(img, uploadUuid string) string {
	return config.ServerPath() + "/" + img + "/blobs/uploads/" + uploadUuid
}

func UploadBlob(img, uploadUuid string, reader io.ReadCloser) (int64, error) {
	fn, err := getBlobUploadFilename(img, uploadUuid)
	if err != nil {
		return 0, err
	}
	return filesys.CopyOrDelete(reader, fn)
}

func PutBlob(img, uploadUuid, digest string, r *http.Request) (len int64, uri string, resultDigest string, err error) {
	len = 0
	uri = ""
	resultDigest = ""
	err = nil

	uploadFn, err := getBlobUploadFilename(img, uploadUuid)
	if err != nil {
		return
	}

	servedFn, err := getBlobServedFilename(img, digest)
	if err != nil {
		return
	}

	resultDigest, err = filesys.CreateDigestFromFile(uploadFn)
	if err != nil || resultDigest != digest {
		filesys.DeleteFile(uploadFn)
		if resultDigest != digest {
			err = errors.New("digest mismatch, expected: " + digest + " got: " + resultDigest)
		}
		return
	}

	err = filesys.RenameOrDelete(uploadFn, servedFn)
	if err != nil {
		return
	}

	len, err = filesys.Size(servedFn)
	if err != nil {
		return
	}

	uri = getBlobServedUri(img, digest, r)

	return
}

func UploadManifest(img, tag string, reader io.ReadCloser) (digest string, modified string, content []byte, err error) {
	digest = ""
	modified = ""
	content = nil
	err = nil

	content, err = io.ReadAll(reader)
	if err != nil {
		return
	}

	digest, err = filesys.CreateDigestFromBuffer(content)
	if err != nil {
		return
	}

	servedFn, err := getManifestServedFilename(img, tag, digest)
	if err != nil {
		return
	}

	servedDir, err := getManifestServedDir(img, tag)
	if err != nil {
		return
	}

	err = filesys.DeleteDir(servedDir)
	if err != nil {
		return
	}

	_, err = filesys.WriteBuffer(servedFn, content)
	if err != nil {
		return
	}

	modified, err = filesys.ModifiedHttpDate(servedFn)
	if err != nil {
		return
	}

	CleanupImage(img)

	return
}

func ExistsManifest(img, tag string) (exists bool, len int64, modified string, digest string) {
	exists = false
	len = -1
	modified = ""
	digest = ""

	servedDir, err := getManifestServedDir(img, tag)
	if err != nil {
		return
	}

	servedFn, err := filesys.GetFirstFilenameInDir(servedDir)
	if err != nil || servedFn == "" {
		return
	}
	digest = fn2digest(servedFn)

	servedFn, err = getManifestServedFilename(img, tag, digest)
	if err != nil {
		return
	}

	len, err = filesys.Size(servedFn)
	if err != nil {
		return
	}

	modified, err = filesys.ModifiedHttpDate(servedFn)
	if err != nil {
		return
	}

	exists = true
	return
}

func DownloadBlob(img, digest string, w http.ResponseWriter) {
	servedFn, err := getBlobServedFilename(img, digest)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	isGzip, err := filesys.IsGzip(servedFn)
	if err != nil {
		log.Error(LOG, "failed to get blob filetype "+servedFn+" err: "+err.Error())
		w.WriteHeader(500)
		return
	}

	contentType := ""
	if isGzip {
		w.Header().Set("Docker-Content-Digest", digest)
		contentType = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	} else {
		contentType = "application/vnd.docker.distribution.manifest.v2+json"
	}

	err = download(servedFn, contentType, w)
	if err != nil {
		log.Error(LOG, "failed to download blob "+servedFn+" err: "+err.Error())
	}
}

func DownloadManifest(img, digest string, w http.ResponseWriter) {
	servedFn, err := findManifestServedFilename(img, digest)
	if err != nil {
		log.Error(LOG, "failed to find manifest "+err.Error())
		w.WriteHeader(500)
		return
	}
	err = download(servedFn, "application/vnd.docker.distribution.manifest.v2+json", w)
	if err != nil {
		log.Error(LOG, "failed to download manifest "+servedFn+" err: "+err.Error())
	}
}

func download(fn, contentType string, w http.ResponseWriter) error {
	len, err := filesys.Size(fn)
	if os.IsNotExist(err) {
		w.WriteHeader(404)
		return err
	}
	if err != nil {
		w.WriteHeader(500)
		return err
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(len, 10))
	w.WriteHeader(200)

	_, err = filesys.Copy(fn, w)
	return err
}

func Cleanup() {
	log.Info(LOG, "cleanup")
	imgs, err := getImages()
	if err != nil {
		log.Error(LOG, "failed to get images")
		return
	}
	for _, img := range imgs {
		CleanupImage(img)
	}
}

func CleanupImage(img string) {
	log.Info(LOG, "cleanup image "+img)

	tags, err := getImageTags(img)
	if err != nil {
		log.Error(LOG, "failed to get image tags")
		return
	}

	var digests = map[string]bool{}

	for _, tag := range tags {
		// log.Info(LOG, "cleanup image "+img+":"+tag)
		manifests, err := getImageManifestFiles(img, tag)
		if err != nil {
			log.Error(LOG, "failed to get image manifests")
			continue
		}
		for _, manifest := range manifests {
			// log.Info(LOG, "cleanup image "+img+":"+tag+" manifest "+manifest)
			manifestJson, err := filesys.ReadJson(manifest)
			if err != nil {
				log.Error(LOG, "failed to read image manifest json "+manifest)
				continue
			}
			if config, ok := manifestJson["config"].(map[string]interface{}); ok {
				digest := config["digest"].(string)
				digests[digest] = true
			} else {
				log.Error(LOG, "failed to get image config from manifest json "+manifest)
				continue
			}
			if layers, ok := manifestJson["layers"].([]interface{}); ok {
				for _, layer := range layers {
					m := layer.(map[string]interface{})
					digest := m["digest"].(string)
					digests[digest] = true
				}
			} else {
				log.Error(LOG, "failed to get image layers from manifest json "+manifest)
				continue
			}
		}
	}

	blobs, err := getImageBlobFiles(img)
	if err != nil {
		log.Error(LOG, "failed to get image blobs")
		return
	}

	for _, blob := range blobs {
		fileDigest := fn2digest(filepath.Base(blob))
		if _, ok := digests[fileDigest]; !ok {
			log.Info(LOG, "deleting orphaned blob "+blob)
			err = filesys.DeleteFile(blob)
			if err != nil {
				log.Error(LOG, "failed to delete orphaned blob "+blob)
			}
		}
	}
}

// repo/v2/imagename/uploads/uploadUid
func getBlobUploadFilename(img, uploadUuid string) (string, error) {
	fn := filepath.Join(config.RepoDir(), config.ServerPath(), img, "uploads", uploadUuid)
	fn, err := filepath.Abs(fn)
	if err != nil {
		return "", err
	}
	return fn, nil
}

// repo/v2/imagename/blobs/digest
func getBlobServedFilename(img, digest string) (string, error) {
	fn := filepath.Join(config.RepoDir(), config.ServerPath(), img, "blobs", digest2fn(digest))
	fn, err := filepath.Abs(fn)
	if err != nil {
		return "", err
	}
	return fn, nil
}

// repo/v2/imagename/blobs/digest
func getBlobServedUrlPath(img, digest string) string {
	return config.ServerPath() + "/" + img + "/blobs/" + digest
}

// https://host[:port]/v2/imagename/blobs/digest
func getBlobServedUri(img, digest string, r *http.Request) string {
	return config.ServerUrl(r) + getBlobServedUrlPath(img, digest)
}

// repo/v2/imagename/manifests/tag
func getManifestServedDir(img, tag string) (string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img, "manifests", tag)
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return dir, nil
}

// repo/v2/imagename/manifests/tag/digest
func getManifestServedFilename(img, tag, digest string) (string, error) {
	dir, err := getManifestServedDir(img, tag)
	if err != nil {
		return "", err
	}
	fn := filepath.Join(dir, digest2fn(digest))
	fn, err = filepath.Abs(fn)
	if err != nil {
		return "", err
	}
	return fn, nil
}

func findManifestServedFilename(img, digest string) (string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img, "manifests")
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filesys.FindFile(dir, digest2fn(digest))
}

func digest2fn(digest string) string {
	return strings.Replace(digest, ":", "-", 1)
}

func fn2digest(fn string) string {
	return strings.Replace(fn, "-", ":", 1)
}

func getImages() ([]string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath())
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return filesys.GetAllFilenamesInDir(dir)
}

func getImageTags(img string) ([]string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img, "manifests")
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return filesys.GetAllFilenamesInDir(dir)
}

func getImageManifestFiles(img, tag string) ([]string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img, "manifests", tag)
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fns, err := filesys.GetAllFilenamesInDir(dir)
	if err != nil {
		return nil, err
	}
	for i, fn := range fns {
		fns[i] = filepath.Join(dir, fn)
	}
	return fns, nil
}

func getImageBlobFiles(img string) ([]string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img, "blobs")
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fns, err := filesys.GetAllFilenamesInDir(dir)
	if err != nil {
		return nil, err
	}
	for i, fn := range fns {
		fns[i] = filepath.Join(dir, fn)
	}
	return fns, nil
}
