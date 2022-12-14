package repo

import (
	"errors"
	"io"
	"io/fs"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/filesys"
	"mosi-docker-registry/pkg/json"
	"mosi-docker-registry/pkg/logging"
	"net/http"
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

	_, err = filesys.WriteBytes(servedFn, content)
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
		logging.Error(LOG, "failed to get blob filetype %s err: %s", servedFn, err.Error())
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
		logging.Error(LOG, "failed to download blob %s err: %s", servedFn, err.Error())
	}
}

func DownloadManifest(img, digest string, w http.ResponseWriter) {
	servedFn, err := findManifestServedFilename(img, digest)
	if errors.Is(err, fs.ErrNotExist) {
		logging.Error(LOG, "manifest not exists %s", err.Error())
		w.WriteHeader(404)
		return
	}
	if err != nil {
		logging.Error(LOG, "failed to find manifest %s", err.Error())
		w.WriteHeader(500)
		return
	}
	err = download(servedFn, "application/vnd.docker.distribution.manifest.v2+json", w)
	if err != nil {
		logging.Error(LOG, "failed to download manifest %s err: %s", servedFn, err.Error())
	}
}

func download(fn, contentType string, w http.ResponseWriter) error {
	len, err := filesys.Size(fn)
	if errors.Is(err, fs.ErrNotExist) {
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

func deleteImage(img, tag string) error {
	dir, err := getManifestServedDir(img, tag)
	if err != nil {
		return err
	}
	return filesys.DeleteDir(dir)
}

func Cleanup() {
	logging.Debug(LOG, "cleanup")
	imgs, err := getImages()
	if err != nil {
		logging.Error(LOG, "cleanup failed to get images")
		return
	}
	for _, img := range imgs {
		CleanupImage(img)
	}
}

func CleanupImage(img string) {
	logging.Debug(LOG, "cleanup image %s", img)

	tags, err := getImageTags(img)
	if err != nil {
		logging.Error(LOG, "cleanup failed to get image tags")
		return
	}

	var digests = map[string]bool{}

	for _, tag := range tags {
		logging.Debug(LOG, "cleanup image %s:%s", img, tag)
		manifestJson, err := getManifestJson(img, tag)
		if err != nil {
			logging.Error(LOG, "cleanup failed to get image manifest json")
			continue
		}

		configDigest, err := getManifestConfigDigest(manifestJson)
		if err != nil {
			logging.Error(LOG, "cleanup failed to get image config digest from manifest json")
			continue
		}

		layerDigests, err := getManifestLayerDigests(manifestJson)
		if err != nil {
			logging.Error(LOG, "cleanup failed to get image layer digests from manifest json")
			continue
		}

		digests[configDigest] = true
		for _, layerDigest := range layerDigests {
			digests[layerDigest] = true
		}
	}

	blobs, err := getBlobFiles(img)
	if err != nil {
		logging.Error(LOG, "cleanup failed to get image blobs %s", img)
		return
	}

	for _, blob := range blobs {
		fileDigest := fn2digest(filepath.Base(blob))
		if _, ok := digests[fileDigest]; !ok {
			logging.Debug(LOG, "deleting orphaned blob %s", blob)
			err = filesys.DeleteFile(blob)
			if err != nil {
				logging.Warn(LOG, "failed to delete orphaned blob %s", blob)
			}
		}
	}

	// check if image has remaining tags, otherwise delete image directory
	tags, err = getImageTags(img)
	if err != nil {
		logging.Error(LOG, "cleanup failed to get image tags")
		return
	}
	if len(tags) == 0 {
		dir, err := getImageServedDir(img)
		if err != nil {
			logging.Error(LOG, "cleanup failed to get image directory")
			return
		}
		logging.Debug(LOG, "deleting image directory %s", dir)
		err = filesys.DeleteDir(dir)
		if err != nil {
			logging.Error(LOG, "cleanup failed to delete image directory %s", dir)
		}
	}
}

func getImageServedDir(img string) (string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img)
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return dir, nil
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

func getManifestFile(img, tag string) (string, error) {
	dir := filepath.Join(config.RepoDir(), config.ServerPath(), img, "manifests", tag)
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	fn, err := filesys.GetFirstFilenameInDir(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fn), nil
}

func getManifestJson(img, tag string) (*json.JsonObject, error) {
	manifest, err := getManifestFile(img, tag)
	if err != nil {
		return nil, err
	}
	return json.DecodeFile(manifest)
}

func getBlobFiles(img string) ([]string, error) {
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

func getManifestConfig(manifestJson *json.JsonObject) (*json.JsonObject, error) {
	if config := manifestJson.GetObject("config", nil); config != nil {
		return config, nil
	}
	return nil, errors.New("failed to get config from manifest")
}

func getManifestConfigDigest(manifestJson *json.JsonObject) (string, error) {
	config, err := getManifestConfig(manifestJson)
	if err != nil {
		return "", nil
	}
	if digest := config.GetString("digest", ""); len(digest) > 0 {
		return digest, nil
	}
	return "", errors.New("failed to get config digest from manifest")
}

func getManifestLayers(manifestJson *json.JsonObject) (*json.JsonArray, error) {
	if layers := manifestJson.GetArray("layers", nil); layers != nil {
		return layers, nil
	}
	return nil, errors.New("failed to get layers from manifest")
}

func getManifestLayer(manifestLayers *json.JsonArray, index int) (*json.JsonObject, error) {
	if layer := manifestLayers.GetObject(index, nil); layer != nil {
		return layer, nil
	}
	return nil, errors.New("failed to get layer from manifest")
}

func getManifestLayerDigest(manifestLayers *json.JsonArray, index int) (string, error) {
	layer, err := getManifestLayer(manifestLayers, index)
	if err != nil {
		return "", err
	}
	if digest := layer.GetString("digest", ""); len(digest) > 0 {
		return digest, nil
	}
	return "", errors.New("failed to get digest from manifest layer")
}

func getManifestLayerDigests(manifestJson *json.JsonObject) ([]string, error) {
	layers, err := getManifestLayers(manifestJson)
	if err != nil {
		return nil, err
	}
	digests := make([]string, layers.Len())
	for i := 0; i < layers.Len(); i++ {
		digest, err := getManifestLayerDigest(layers, i)
		if err != nil {
			return nil, err
		}
		digests[i] = digest
	}
	return digests, nil
}
