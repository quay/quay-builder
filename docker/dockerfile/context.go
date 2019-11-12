package dockerfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/tarsum"
)

const unix1980 int64 = 315532800

// loadBuildContext loads the build context into a tarsum.
func loadBuildContext(buildContextDirectory string) (tarsum.TarSum, error) {
	// Zero out mtimes so that they don't effect the outcome of our tarsum.
	err := chtimesDir(buildContextDirectory)
	if err != nil {
		return nil, err
	}

	// Compress our build context directory into a tar.
	tarred, err := archive.Tar(buildContextDirectory, archive.Uncompressed)
	if err != nil {
		return nil, err
	}

	// Create a tarsum of the tar.
	buildContext, err := tarsum.NewTarSum(tarred, true, tarsum.Version0)
	if err != nil {
		return nil, err
	}

	// Create a temporary directory.
	tmpdirPath, err := ioutil.TempDir("", "docker-build")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpdirPath)

	// Extract the tar to the temporary directory.
	// This is required in order for the tarsum to be calculated.
	if err := archive.Untar(buildContext, tmpdirPath, nil); err != nil {
		return nil, err
	}

	return buildContext, nil
}

// chtimesDir walks a directory and sets the atime and mtime to 1 Jan 1980 --
// the earliest timestamp supported by zip.
func chtimesDir(root string) error {
	return filepath.Walk(root, walkFunc)
}

func walkFunc(path string, info os.FileInfo, err error) error {
	// This handles any errors we got walking the filesystem.
	if err != nil {
		return err
	}

	err = os.Chtimes(path, time.Unix(unix1980, 0), time.Unix(unix1980, 0))

	// If we can't find the file we just walked to, it's a broken symlink.
	// Just ignore these.
	if err != nil && strings.Contains(err.Error(), "no such file or directory") {
		err = nil
	}

	return err
}
