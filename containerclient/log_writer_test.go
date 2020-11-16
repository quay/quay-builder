package containerclient

import (
	"bytes"
	"encoding/json"
	"testing"
)

type testWriter struct {
	buf         bytes.Buffer
	errResponse *Response
}

func (w testWriter) Write(p []byte) (int, error) {
	resp := Response{}
	json.Unmarshal(p, &resp)
	if resp.Error != "" {
		w.errResponse = &resp
	}

	return w.buf.Write(p)
}

func (w testWriter) HasErrResponse() bool {
	return false
}

func (w testWriter) ErrResponse() *Response {
	return nil
}

func TestPartialBufferHasContents(t *testing.T) {
	table := []struct {
		buf      []byte
		expected bool
	}{
		{[]byte{}, false},
		{[]byte{0x1, 0x2}, true},
	}

	for _, tt := range table {
		pbuf := partialBuffer(tt.buf)
		got := pbuf.hasContents()
		if got != tt.expected {
			t.Errorf("want: %v, got: %v", tt.expected, got)
		}
	}
}

func TestPartialBufferGetAndEmpty(t *testing.T) {
	table := [][]byte{
		{},
		{0x1, 0x2},
	}

	for _, tt := range table {
		og := make([]byte, len(tt))
		copy(og, tt)
		pbuf := partialBuffer(tt)

		got := pbuf.getAndEmpty([]byte{})

		if !bytes.Equal(got, og) {
			t.Errorf("want: %v, got: %v", og, got)
		}

		if len(pbuf) != 0 {
			t.Errorf("failed to empty %v", og)
		}
	}
}

func TestPartialBufferSet(t *testing.T) {
	table := []struct {
		buf      []byte
		expected partialBuffer
	}{
		{[]byte{}, partialBuffer([]byte{})},
		{[]byte{0x1, 0x2}, partialBuffer([]byte{0x1, 0x2})},
	}

	for _, tt := range table {
		var pbuf partialBuffer
		pbuf.set(tt.buf)
		if !bytes.Equal(pbuf, tt.expected) {
			t.Errorf("want: %v, got: %v", tt.expected, pbuf)
		}
	}
}
