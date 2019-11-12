package docker

import "github.com/fsouza/go-dockerclient"

type TestDockerClient struct {
	err          error
	ImagePulled  bool
	ImagePushed  bool
	ImageTagged  bool
	ImageBuilt   bool
	ImageRemoved bool
}

func newTestDockerClient(err error) Client {
	return &TestDockerClient{
		err: err,
	}
}

func (c *TestDockerClient) BuildImage(docker.BuildImageOptions) error {
	c.ImageBuilt = true
	return c.err
}

func (c *TestDockerClient) PullImage(docker.PullImageOptions, docker.AuthConfiguration) error {
	c.ImagePulled = true
	return c.err
}

func (c *TestDockerClient) PushImage(docker.PushImageOptions, docker.AuthConfiguration) error {
	c.ImagePushed = true
	return c.err
}

func (c *TestDockerClient) TagImage(string, docker.TagImageOptions) error {
	c.ImageTagged = true
	return c.err
}

func (c *TestDockerClient) InspectImage(string) (*docker.Image, error) {
	return &docker.Image{ID: ""}, c.err
}

func (c *TestDockerClient) PruneImages(docker.PruneImagesOptions) (*docker.PruneImagesResults, error) {
	return &docker.PruneImagesResults{}, c.err
}

func (c *TestDockerClient) RemoveImageExtended(string, docker.RemoveImageOptions) error {
	c.ImageRemoved = true
	return c.err
}
