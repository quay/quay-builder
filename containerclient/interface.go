package containerclient

import (
	"fmt"
	"io"
	"strings"
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

func NewClient(host, runtime string) (Client, error) {
	runtime = strings.ToLower(runtime)
	if runtime != "docker" && runtime != "podman" {
		return nil, fmt.Errorf("Invalid container runtime: %s", runtime)
	}

	if runtime == "docker" {
		return NewDockerClient(host)
	} else {
		return NewPodmanClient(host)
	}
}
