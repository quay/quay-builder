package containerclient

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/rpc"
)

// PodmanRPCWriter implements a RPCWriter.
// Unlike the Docker daemon, Podman's build call outputs plain string, and not JSON encoded data,
// so we need to serialize each line into a Response struct before logging it to an rpc.Client.
type PodmanRPCWriter struct {
	client           rpc.Client
	errResponse      *Response
	partialBuffer    *partialBuffer
	hasPartialBuffer bool
}

// Write implements the io.Writer interface for RPCWriter.
func (w *PodmanRPCWriter) Write(p []byte) (n int, err error) {
	// Unlike docker, libpod parses the JSON encoded data from stream before writing the output,
	// without the option of returning the raw data instead.
	// Instead of decoding the stream into a Response, we set the Response's "Stream" before
	// marshaling it into JSON to be logged.
	originalLength := len(p)

	var m Response
	m.Stream = string(p)

	jsonData, err := json.Marshal(&m)
	if err != nil {
		log.Fatalf("Error when marshaling logs: %v", err)
	}

	err = w.client.PublishBuildLogEntry(string(jsonData))
	if err != nil {
		log.Fatalf("Failed to publish log entry: %v", err)
	}

	return originalLength, nil
}

func (w *PodmanRPCWriter) ErrResponse() (error, bool) {
	// libpod already parses the JSON stream before writing to output.
	// So the error would not be returned from the output stream,. but as
	// the return value of the API call instead.
	// See https://github.com/containers/podman/blob/master/pkg/bindings/images/build.go#L175
	return nil, false
}

func (w *PodmanRPCWriter) ResetError() {
	w.errResponse = nil
}
