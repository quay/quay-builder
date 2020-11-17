package containerclient

import (
	"io"

	"github.com/fsouza/go-dockerclient"
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
	BuildImage(docker.BuildImageOptions) error
	PullImage(docker.PullImageOptions, docker.AuthConfiguration) error
	PushImage(docker.PushImageOptions, docker.AuthConfiguration) error
	TagImage(string, docker.TagImageOptions) error
	InspectImage(string) (*docker.Image, error)
	RemoveImageExtended(string, docker.RemoveImageOptions) error
	PruneImages(docker.PruneImagesOptions) (*docker.PruneImagesResults, error)
}
