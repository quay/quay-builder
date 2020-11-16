package dockerfile

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"
	"syscall"

	"github.com/docker/docker/pkg/tarsum"
)

// quoteSlice calls strconv.Quote on each of the elements in the slice, returning a new
// slice with the values.
func quoteSlice(slice []string) []string {
	var parts = make([]string, len(slice))
	for index := range slice {
		parts[index] = strconv.Quote(slice[index])
	}
	return parts
}

// fileHash calculates the hash of a standalone regular file for a
// given tarsum version.
func fileHash(v tarsum.Version, name string) (string, error) {
	if v != tarsum.Version0 {
		return "", errors.New("fileHash: unsupported tarsum version")
	}

	f, err := os.Open(name)
	if err != nil {
		return "", err
	}

	defer f.Close()

	fi, err := os.Stat(name)
	if err != nil {
		return "", err
	}

	if !fi.Mode().IsRegular() {
		return "", errors.New("fileHash called on non-regular file")
	}

	fih, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return "", err
	}

	header := []string{
		"name", fih.Name,
		"mode", strconv.FormatInt(fih.Mode|syscall.S_IFREG, 10),
		"uid", strconv.Itoa(fih.Uid),
		"gid", strconv.Itoa(fih.Gid),
		"size", strconv.FormatInt(fih.Size, 10),
		"mtime", strconv.FormatInt(fih.ModTime.UTC().Unix(), 10),
		"typeflag", string([]byte{fih.Typeflag}),
		"linkname", fih.Linkname,
		"uname", fih.Uname,
		"gname", fih.Gname,
		"devmajor", strconv.FormatInt(fih.Devmajor, 10),
		"devminor", strconv.FormatInt(fih.Devminor, 10),
	}

	// Could get this from tarsum.DefaultTHash, but this seems fine.
	h := sha256.New()

	// Write the header to the hash function.
	for _, s := range header {
		h.Write([]byte(s))
	}

	// Followed by the body of the file.
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
