package dockerfile

import (
	"os"
	"testing"

	"github.com/docker/docker/pkg/tarsum"
)

var mobyHash string

func TestMain(m *testing.M) {
	var err error

	// This needs to happen before using fileHash().
	chtimesDir("testdata")

	mobyHash, err = fileHash(tarsum.Version0, "testdata/moby.txt")
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
