package containerclient

import (
	"io"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/rpc"
)

type BuildImageOptions struct {
	// name:tag
	Name                string
	NoCache             bool
	CacheFrom           []string
	SuppressOutput      bool
	RmTmpContainer      bool
	ForceRmTmpContainer bool
	OutputStream        io.Writer
	Dockerfile          string
	ContextDir          string
}

type AuthConfiguration struct {
	Username string
	Password string
}

type PullImageOptions struct {
	Repository   string
	Registry     string
	Tag          string
	OutputStream io.Writer
}

type PushImageOptions struct {
	Repository   string
	Registry     string
	Tag          string
	OutputStream io.Writer
}

type TagImageOptions struct {
	Repository string
	Tag        string
	Force      bool
}

type Image struct {
	ID          string
	RepoDigests []string
}

type RemoveImageOptions struct {
	Force bool
}

type PruneImagesOptions struct {
	Filters map[string][]string
}

type PruneImagesResults struct {
	ImagesDeleted  []string
	SpaceReclaimed int64
}

// Client is an interface for all of the container/image interactions required of a
// worker. This includes Docker and/or Podman
type Client interface {
	BuildImage(BuildImageOptions) error
	PullImage(PullImageOptions, AuthConfiguration) error
	PushImage(PushImageOptions, AuthConfiguration) error
	TagImage(string, TagImageOptions) error
	InspectImage(string) (*Image, error)
	RemoveImageExtended(string, RemoveImageOptions) error
	PruneImages(PruneImagesOptions) (*PruneImagesResults, error)
}

func NewClient(host, containerRuntime string) (Client, error) {
	containerRuntime = strings.ToLower(containerRuntime)
	if containerRuntime != "docker" && containerRuntime != "podman" {
		log.Fatal("Invalid container runtime:", containerRuntime)
	}

	if containerRuntime == "docker" {
		return NewDockerClient(host)
	} else {
		return NewPodmanClient(host)
	}
}

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

// NewRPCWriter allocates a new Writer that streams logs via an RPC client.
func NewRPCWriter(client rpc.Client, containerRuntime string) LogWriter {
	containerRuntime = strings.ToLower(containerRuntime)
	if containerRuntime != "docker" && containerRuntime != "podman" {
		log.Fatal("Invalid container runtime:", containerRuntime)
	}

	if containerRuntime == "docker" {
		return &DockerRPCWriter{
			client:        client,
			partialBuffer: new(partialBuffer),
		}
	} else {
		return &PodmanRPCWriter{
			client:        client,
			partialBuffer: new(partialBuffer),
		}
	}

}

// Response represents a response from a Docker™ daemon or podman.
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
