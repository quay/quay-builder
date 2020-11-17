package containerclient

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

func (c *TestDockerClient) BuildImage(BuildImageOptions) error {
	c.ImageBuilt = true
	return c.err
}

func (c *TestDockerClient) PullImage(PullImageOptions, AuthConfiguration) error {
	c.ImagePulled = true
	return c.err
}

func (c *TestDockerClient) PushImage(PushImageOptions, AuthConfiguration) error {
	c.ImagePushed = true
	return c.err
}

func (c *TestDockerClient) TagImage(string, TagImageOptions) error {
	c.ImageTagged = true
	return c.err
}

func (c *TestDockerClient) InspectImage(string) (*Image, error) {
	return &Image{ID: ""}, c.err
}

func (c *TestDockerClient) PruneImages(PruneImagesOptions) (*PruneImagesResults, error) {
	return &PruneImagesResults{}, c.err
}

func (c *TestDockerClient) RemoveImageExtended(string, RemoveImageOptions) error {
	c.ImageRemoved = true
	return c.err
}
