package docker

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/rpc"
)

const (
	minProgressDelta = 10000000
	bufferingStatus  = "Buffering to disk"
	pushingStatus    = "Pushing"
)

// LogWriter represents anything that can stream Docker logs from the daemon
// and check if any error has occured.
type LogWriter interface {
	// ErrResponse returns an error that occurred from Docker and resets the
	// state of that internal error value to nil. If there is no error, returns
	// false as part of the tuple.
	ErrResponse() (error, bool)

	// ResetError throws away any error state from previously streamed logs.
	ResetError()

	io.Writer
}

// partialBuffer represents a buffer of data that was unable to be previously
// serialized because it was not enough data was provided to form valid JSON.
type partialBuffer []byte

func (pb partialBuffer) hasContents() bool { return len(pb) != 0 }

func (pb *partialBuffer) set(in []byte) { *pb = in }

func (pb *partialBuffer) getAndEmpty(in []byte) (ret []byte) {
	ret = append(ret, *pb...)
	ret = append(ret, in...)

	*pb = []byte{}
	return
}

// RPCWriter implements a Writer that consumes encoded JSON data and buffers it
// until it has a valid JSON object and then logs it to an rpc.Client.
type RPCWriter struct {
	client           rpc.Client
	errResponse      *Response
	partialBuffer    *partialBuffer
	hasPartialBuffer bool
}

// NewRPCWriter allocates a new Writer that streams logs via an RPC client.
func NewRPCWriter(client rpc.Client) LogWriter {
	return &RPCWriter{
		client:        client,
		partialBuffer: new(partialBuffer),
	}
}

// Write implements the io.Writer interface for RPCWriter.
func (w *RPCWriter) Write(p []byte) (n int, err error) {
	originalLength := len(p)

	// Note: Sometimes Docker returns to us only the beginning of a stream,
	// so we have to prepend any existing data from the previous call.
	if w.partialBuffer.hasContents() {
		p = w.partialBuffer.getAndEmpty(p)
	}

	buf := bytes.NewBuffer(p)
	dec := json.NewDecoder(buf)
	f := &filter{}

	for {
		// Yield to the Go scheduler. Sometimes, when we have very large number of
		// messages, we need to yield to ensure that other goroutines are not
		// starved (specifically the heartbeat).
		runtime.Gosched()

		// Attempt to decode what was written into a Docker Reponse.
		var m Response
		if err = dec.Decode(&m); err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			// If we get an unexpected EOF, it means that the JSON response from
			// Docker was too large to fit into the single Write call. Therefore, we
			// store any unparsed data and prepend it on the next call.
			var bufferedData []byte
			bufferedData, err = ioutil.ReadAll(dec.Buffered())
			if err != nil {
				log.Fatalf("Error when reading buffered logs: %v", err)
			}
			w.partialBuffer.set(bufferedData)
			break
		} else if err != nil {
			// Try to determine what we failed to decode.
			entry, readErr := ioutil.ReadAll(dec.Buffered())
			if readErr != nil {
				entry = []byte("unknown")
			}
			log.Fatalf("Error when reading logs: %v; Failed entry: %v", err, string(entry))
		}

		if m.Error != "" {
			w.errResponse = &m
			continue
		}

		if f.shouldSkip(&m) {
			continue
		}

		jsonData, err := json.Marshal(&m)
		if err != nil {
			log.Fatalf("Error when marshaling logs: %v", err)
		}

		err = w.client.PublishBuildLogEntry(string(jsonData))
		if err != nil {
			log.Fatalf("Failed to publish log entry: %v", err)
		}
	}

	return originalLength, nil
}

// ErrResponse returns an error that occurred from Docker and then calls
// ResetError().
func (w *RPCWriter) ErrResponse() (error, bool) {
	err := w.errResponse
	w.ResetError()

	if err == nil {
		return nil, false
	}

	return errors.New(err.Error), true
}

// ResetError throws away any error state from previously streamed logs.
func (w *RPCWriter) ResetError() {
	w.errResponse = nil
}

// Response represents a response from a Docker™ daemon.
type Response struct {
	Error          string         `json:"error,omitempty"`
	Stream         string         `json:"stream,omitempty"`
	Status         string         `json:"status,omitempty"`
	ID             string         `json:"id,omitempty"`
	ProgressDetail progressDetail `json:"progressDetail,omitempty"`
}

// progressDetail represents the progress made by a Docker™ command.
type progressDetail struct {
	Current int `json:"current,omitempty"`
	Total   int `json:"total,omitempty"`
}

type filter struct {
	lastSent *Response
}

func (f filter) shouldSkip(resp *Response) bool {
	if f.lastSent == nil {
		f.lastSent = resp
		return false
	}

	// Don't send the response if it hasn't transfered the minimum amount across
	// the docker socket.
	if resp.Status == bufferingStatus && f.lastSent.Status == bufferingStatus {
		switch {
		case resp.ProgressDetail.Current < f.lastSent.ProgressDetail.Current+minProgressDelta:
			return true
		default:
			return false
		}
	}

	// Don't send the push response unless it has pushed more than the minimum.
	if resp.Status == pushingStatus && f.lastSent.Status == pushingStatus {
		switch {
		case resp.ProgressDetail.Current == f.lastSent.ProgressDetail.Total:
			// Always send the final response.
			return false
		case resp.ProgressDetail.Current < f.lastSent.ProgressDetail.Current+minProgressDelta:
			return true
		default:
			return false
		}
	}

	return false
}
