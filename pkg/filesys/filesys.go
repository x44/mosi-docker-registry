package filesys

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
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

func IsGzip(fn string) (bool, error) {
	f, err := os.Open(fn)
	if err != nil {
		return false, err
	}
	defer f.Close()
	b := []byte{0, 0, 0}
	_, err = f.Read(b)
	if err != nil {
		return false, err
	}
	return b[0] == 0x1f && b[1] == 0x8b && b[2] == 0x08, nil
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

func Copy(fn string, w io.Writer) (int64, error) {
	f, err := os.Open(fn)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	return io.Copy(w, f)
}

func DeleteFile(fn string) error {
	return os.Remove(fn)
}

func DeleteDir(dir string) error {
	return os.RemoveAll(dir)
}

func RenameOrDelete(src, dst string) error {
	dir := filepath.Dir(dst)
	err := CreateDir(dir)
	if err != nil {
		DeleteFile(src)
		return err
	}
	err = os.Rename(src, dst)
	if err != nil {
		DeleteFile(src)
		DeleteFile(dst)
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
		DeleteFile(fn)
		return -1, err
	}

	err = CloseFileOrDelete(f)
	return len, err
}

func GetFirstFilenameInDir(dir string) (string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fns, err := f.Readdirnames(1)
	if err != nil {
		return "", nil
	}
	if len(fns) == 0 {
		return "", nil
	}
	return fns[0], nil
}

func GetAllFilenamesInDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fns, err := f.Readdirnames(-1)
	if err != nil {
		return nil, nil
	}
	return fns, nil
}

// depth-first
func FindFile(dir, fn string) (string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	fns, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return "", nil
	}
	for _, child := range fns {
		joined := filepath.Join(dir, child)
		var stat fs.FileInfo = nil
		isDir := true
		if child == fn {
			stat, err = os.Stat(joined)
			if err != nil {
				return "", err
			}
			isDir = stat.IsDir()
			if !isDir {
				return joined, nil
			}
		}
		// Note that if stat != nil then isDir is already valid
		if stat == nil {
			stat, err = os.Stat(joined)
			if err != nil {
				return "", err
			}
			isDir = stat.IsDir()
		}
		if isDir {
			sub, err := FindFile(joined, fn)
			if sub != "" {
				return sub, err
			}
		}
	}
	return "", nil
}

func ReadJson(fn string) (map[string]any, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m := map[string]any{}
	err = json.NewDecoder(f).Decode(&m)
	if err != nil {
		return nil, err
	}
	return m, nil
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
