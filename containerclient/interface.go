package containerclient

import (
	"github.com/fsouza/go-dockerclient"
)

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
