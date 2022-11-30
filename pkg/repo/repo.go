package repo

import (
	"docker-repo/pkg/config"
	"docker-repo/pkg/filesys"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

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

func PutBlob(img, uploadUuid, digest string) (len int64, uri string, resultDigest string, err error) {
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
		filesys.Delete(uploadFn)
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

	uri = getBlobServedUri(img, digest)

	return
}

func UploadManifest(img, version string, reader io.ReadCloser) (digest string, modified string, content []byte, err error) {
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

	servedFn, err := getManifestServedFilename(img, version, digest)
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
	return
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
	digest = strings.Replace(digest, ":", "-", 1)
	fn := filepath.Join(config.RepoDir(), config.ServerPath(), img, "blobs", digest)
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
func getBlobServedUri(img, digest string) string {
	return config.ServerAddress() + getBlobServedUrlPath(img, digest)
}

// repo/v2/imagename/manifests/version/digest
func getManifestServedFilename(img, version, digest string) (string, error) {
	digest = strings.Replace(digest, ":", "-", 1)
	fn := filepath.Join(config.RepoDir(), config.ServerPath(), img, "manifests", version, digest)
	fn, err := filepath.Abs(fn)
	if err != nil {
		return "", err
	}
	return fn, nil
}
