package filesys

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var gmtTimeLoc = time.FixedZone("GMT", 0)

func Size(fn string) (int64, error) {
	fileInfo, err := os.Stat(fn)
	if err != nil {
		return -1, err
	}
	return fileInfo.Size(), nil
}

// "Tue, 29 Nov 2022 14:56:29 GMT"
func ModifiedHttpDate(fn string) (string, error) {
	fileInfo, err := os.Stat(fn)
	if err != nil {
		return "", err
	}
	return fileInfo.ModTime().In(gmtTimeLoc).Format(http.TimeFormat), nil
}

func CreateDir(dir string) error {
	fileInfo, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}

	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return errors.New(dir + " is not a directory")
	}
	return nil
}

func CreateFile(fn string) (*os.File, error) {
	dir := filepath.Dir(fn)
	err := CreateDir(dir)
	if err != nil {
		return nil, err
	}
	return os.Create(fn)
}

func CloseFileOrDelete(f *os.File) error {
	err := f.Close()
	if err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}

func CopyOrDelete(src io.Reader, dst string) (int64, error) {
	f, err := CreateFile(dst)
	if err != nil {
		return -1, err
	}

	written, err := io.Copy(f, src)

	if err != nil {
		os.Remove(f.Name())
		return -1, err
	}

	err = CloseFileOrDelete(f)

	if err != nil {
		return -1, err
	}
	return written, nil
}

func Delete(fn string) error {
	return os.Remove(fn)
}

func RenameOrDelete(src, dst string) error {
	dir := filepath.Dir(dst)
	err := CreateDir(dir)
	if err != nil {
		Delete(src)
		return err
	}
	err = os.Rename(src, dst)
	if err != nil {
		Delete(src)
		Delete(dst)
	}
	return err
}

func WriteBuffer(fn string, data []byte) (int, error) {
	f, err := CreateFile(fn)
	if err != nil {
		return -1, err
	}

	len, err := f.Write(data)
	if err != nil {
		Delete(fn)
		return -1, err
	}

	err = CloseFileOrDelete(f)
	return len, err
}

func CreateDigestFromFile(fn string) (string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sha := sha256.New()
	if _, err := io.Copy(sha, f); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(sha.Sum(nil)), nil
}

func CreateDigestFromBuffer(buf []byte) (string, error) {
	sha := sha256.New()
	if _, err := sha.Write(buf); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sha.Sum(nil)), nil
}
