package repo

import (
	"errors"
	"io"
	"io/fs"
	"mosi-docker-registry/pkg/config"
	"mosi-docker-registry/pkg/filesys"
	"mosi-docker-registry/pkg/json"
	"mosi-docker-registry/pkg/logging"
	"mosi-docker-registry/pkg/wildcard"
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
		// TODO
		if config, ok := manifestJson["config"].(map[string]interface{}); ok {
			digest := config["digest"].(string)
			digests[digest] = true
		} else {
			logging.Error(LOG, "cleanup failed to get image config from manifest json")
			continue
		}
		if layers, ok := manifestJson["layers"].([]interface{}); ok {
			for _, layer := range layers {
				m := layer.(map[string]interface{})
				digest := m["digest"].(string)
				digests[digest] = true
			}
		} else {
			logging.Error(LOG, "cleanup failed to get image layers from manifest json")
			continue
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

func getManifestJson(img, tag string) (map[string]any, error) {
	manifest, err := getManifestFile(img, tag)
	if err != nil {
		return nil, err
	}
	return filesys.ReadJson(manifest)
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

func getManifestConfig(manifestJson *map[string]any) (*map[string]interface{}, error) {
	if config, ok := (*manifestJson)["config"].(map[string]interface{}); ok {
		return &config, nil
	}
	return nil, errors.New("failed to get config from manifest")
}

func getManifestConfigDigest(manifestJson *map[string]any) (string, error) {
	config, err := getManifestConfig(manifestJson)
	if err != nil {
		return "", nil
	}
	if digest, ok := (*config)["digest"].(string); ok {
		return digest, nil
	}
	return "", errors.New("failed to get config digest from manifest")
}

func getManifestLayers(manifestJson *map[string]any) (*[]interface{}, error) {
	if layers, ok := (*manifestJson)["layers"].([]interface{}); ok {
		return &layers, nil

	}
	return nil, errors.New("failed to get layers from manifest")
}

func getManifestLayer(manifestLayers *[]interface{}, index int) (*map[string]interface{}, error) {
	if layer, ok := (*manifestLayers)[index].(map[string]interface{}); ok {
		return &layer, nil
	}
	return nil, errors.New("failed to get layer from manifest")
}

func getManifestLayerDigest(manifestLayers *[]interface{}, index int) (string, error) {
	layer, err := getManifestLayer(manifestLayers, index)
	if err != nil {
		return "", err
	}
	if digest, ok := (*layer)["digest"].(string); ok {
		return digest, nil
	}
	return "", errors.New("failed to get digtest from manifest layer")
}

func getManifestLayerDigests(manifestJson *map[string]any) ([]string, error) {
	manifestLayers, err := getManifestLayers(manifestJson)
	if err != nil {
		return nil, err
	}
	var digests = make([]string, len(*manifestLayers))
	for i := 0; i < len(*manifestLayers); i++ {
		digest, err := getManifestLayerDigest(manifestLayers, i)
		if err != nil {
			return nil, err
		}
		digests[i] = digest
	}
	return digests, nil
}

////////////////////////////////////////////////////////////////////////////////
// CLI
////////////////////////////////////////////////////////////////////////////////

func List(imgPattern, tagPattern string) (*json.JsonObject, error) {
	if tagPattern == "" {
		return listImages(imgPattern)
	} else {
		return listLayers(imgPattern, tagPattern)
	}
}

func listImages(imgPattern string) (*json.JsonObject, error) {
	imgs, err := getImages()
	if err != nil {
		return nil, err
	}

	tables := json.NewJsonArray(0)
	res := json.NewJsonObject()
	res.Put("tables", tables)

	var table *json.JsonObject = nil
	var rows *json.JsonArray = nil

	for _, img := range imgs {
		if wildcard.Matches(img, imgPattern) {

			if table == nil {
				table = json.NewJsonObject()
				table.Put("fields", json.JsonArrayFromStrings("Image", "Tags", "Blobs", "Size"))
				tables.Add(table)

				rows = json.NewJsonArray(0)
				table.Put("rows", rows)
			}

			nTags := -1
			tags, err := getImageTags(img)
			if err != nil {
				return nil, err
			}
			nTags = len(tags)

			nBlobs := -1
			var nBlobBytes int64 = 0
			blobs, err := getBlobFiles(img)
			if err != nil {
				return nil, err
			}
			nBlobs = len(blobs)

			for _, blob := range blobs {
				size, err := filesys.Size(blob)
				if err != nil {
					return nil, err
				}
				nBlobBytes += size
			}
			rows.Add(json.JsonArrayFromAny(img, nTags, nBlobs, filesys.Bytes2IEC(nBlobBytes)))
		}
	}

	return res, nil
}

func listLayers(imgPattern, tagPattern string) (*json.JsonObject, error) {
	imgs, err := getImages()
	if err != nil {
		return nil, err
	}

	tables := json.NewJsonArray(0)
	res := json.NewJsonObject()
	res.Put("tables", tables)

	for _, img := range imgs {
		if wildcard.Matches(img, imgPattern) {

			tags, err := getImageTags(img)
			if err != nil {
				return nil, err
			}
			for _, tag := range tags {
				if wildcard.Matches(tag, tagPattern) {

					table := json.NewJsonObject()
					tables.Add(table)
					table.Put("fields", json.JsonArrayFromStrings("Image", "Tag", "Layer", "Size"))
					rows := json.NewJsonArray(0)
					table.Put("rows", rows)

					manifestJson, err := getManifestJson(img, tag)
					if err != nil {
						return nil, err
					}

					// configDigest, err := getManifestConfigDigest(&manifestJson)
					// if err != nil {
					// 	return nil, err
					// }

					layerDigests, err := getManifestLayerDigests(&manifestJson)
					if err != nil {
						return nil, err
					}

					for _, layerDigest := range layerDigests {
						servedBlobFn, err := getBlobServedFilename(img, layerDigest)
						if err != nil {
							return nil, err
						}
						nLayerBytes, err := filesys.Size(servedBlobFn)
						if err != nil {
							return nil, err
						}

						rows.Add(json.JsonArrayFromStrings(img, tag, layerDigest, filesys.Bytes2IEC(nLayerBytes)))
					}
				}
			}

			// nBlobs := -1
			// var nBlobBytes int64 = 0
			// blobs, err := getBlobFiles(img)
			// if err == nil {
			// 	nBlobs = len(blobs)

			// 	for _, blob := range blobs {
			// 		size, err := filesys.Size(blob)
			// 		if err == nil {
			// 			nBlobBytes += size
			// 		}
			// 	}
			// }
		}
	}

	return res, nil
}
