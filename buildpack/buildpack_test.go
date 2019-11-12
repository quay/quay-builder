package buildpack

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var (
	gzippedDockerfile = []byte{31, 139, 8, 0, 19, 247, 182, 84, 0, 3, 237, 207, 187, 10, 194, 64, 16, 133, 225, 212, 62, 197, 130, 141, 118, 179, 183, 44, 88, 251, 34, 34, 187, 176, 222, 34, 201, 230, 253, 141, 46, 104, 23, 108, 130, 8, 255, 215, 28, 134, 153, 226, 204, 190, 59, 158, 99, 159, 242, 37, 54, 139, 17, 145, 214, 57, 245, 204, 208, 250, 87, 138, 169, 115, 165, 189, 210, 198, 121, 239, 131, 53, 198, 42, 209, 86, 187, 208, 40, 89, 174, 210, 199, 56, 148, 67, 63, 85, 57, 229, 107, 158, 187, 155, 206, 82, 154, 217, 215, 79, 212, 59, 255, 196, 122, 115, 235, 238, 219, 157, 42, 113, 40, 171, 95, 151, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 124, 237, 1, 92, 183, 98, 255, 0, 40, 0, 0}

	zippedDockerfile = []byte{80, 75, 3, 4, 10, 0, 0, 0, 0, 0, 224, 144, 46, 70, 37, 169, 13, 221, 13, 0, 0, 0, 13, 0, 0, 0, 10, 0, 28, 0, 68, 111, 99, 107, 101, 114, 102, 105, 108, 101, 85, 84, 9, 0, 3, 147, 246, 182, 84, 147, 246, 182, 84, 117, 120, 11, 0, 1, 4, 245, 1, 0, 0, 4, 20, 0, 0, 0, 35, 40, 110, 111, 112, 41, 58, 32, 116, 101, 115, 116, 10, 80, 75, 1, 2, 30, 3, 10, 0, 0, 0, 0, 0, 224, 144, 46, 70, 37, 169, 13, 221, 13, 0, 0, 0, 13, 0, 0, 0, 10, 0, 24, 0, 0, 0, 0, 0, 1, 0, 0, 0, 164, 129, 0, 0, 0, 0, 68, 111, 99, 107, 101, 114, 102, 105, 108, 101, 85, 84, 5, 0, 3, 147, 246, 182, 84, 117, 120, 11, 0, 1, 4, 245, 1, 0, 0, 4, 20, 0, 0, 0, 80, 75, 5, 6, 0, 0, 0, 0, 1, 0, 1, 0, 80, 0, 0, 0, 81, 0, 0, 0, 0, 0}

	dockerfileContents = []byte("#(nop): test dockerfile")

	extractTests = []struct {
		body              []byte
		mimetype          string
		expectedFileCount int
	}{
		{zippedDockerfile, "application/x-zip-compressed", 2},
		{gzippedDockerfile, "application/gzip", 2},
		{gzippedDockerfile, "application/x-gzip", 2},
		{dockerfileContents, "application/octet-stream", 2},
		{dockerfileContents, "text/plain", 2},
	}
)

func TestExtractBuildPackage(t *testing.T) {
	for _, tt := range extractTests {
		path, err := extractBuildPackage(bytes.NewBuffer(tt.body), tt.mimetype)
		if err != nil {
			t.Fatal(err)
		}

		fileCount := 0
		filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
			fileCount++
			return nil
		})

		if fileCount != tt.expectedFileCount {
			t.Fatalf("Unexpected file count in directory: %s", path)
		}
	}
}

// TestDownload tests that an empty Content-Type will auto-detect the mimetype
// based on the first bytes of the body.
func TestDownload(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "")
		w.Write(gzippedDockerfile)
	}))
	defer s.Close()
	path, err := download(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadFile(filepath.Join(path, "Dockerfile"))
	if err != nil {
		t.Error(err)
	} else if string(b) != "#(nop): test\n" {
		t.Errorf("unexpected contents: %s", b)
	}
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}
