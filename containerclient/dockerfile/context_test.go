package dockerfile

import (
	"errors"
	"os"
	"testing"
)

func TestWalkFunc(t *testing.T) {
	table := []struct {
		path   string
		info   os.FileInfo
		err    error
		result error
	}{
		{"", nil, errors.New("boom"), errors.New("boom")},
	}

	for _, tt := range table {
		result := walkFunc(tt.path, tt.info, tt.err)
		if result.Error() != tt.result.Error() {
			t.Errorf("got %v, wanted %v", result, tt.result)
		}
	}
}
