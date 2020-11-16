package podmanclient

import (
	"context"
	"net/url"

	"github.com/containers/podman/v2/pkg/bindings"

	"github.com/quay/quay-builder/containerclient"
)

type podmanClient struct {
	podmanContext context.Context
}

func NewClient(host string) (*podmanClient, error) {
	hostURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	// Podman's connection context.
	// This must be passed to every api calls
	pmContext, err := bindings.NewConnection(context.Background(), hostURL.String())
	if err != nil {
		return nil, err
	}
	c := &podmanClient{
		podmanContext: pmContext,
	}

	return c, nil
}

func (c *podmanClient) BuildImage(containerclient.BuildImageOptions) error {
	return nil
}

func (c *podmanClient) PullImage(containerclient.PullImageOptions, containerclient.AuthConfiguration) error {
	return nil
}

func (c *podmanClient) PushImage(containerclient.PushImageOptions, containerclient.AuthConfiguration) error {
	return nil
}

func (c *podmanClient) TagImage(string, containerclient.TagImageOptions) error {
	return nil
}

func (c *podmanClient) InspectImage(string) (*containerclient.Image, error) {
	return nil, nil
}

func (c *podmanClient) RemoveImageExtended(string, containerclient.RemoveImageOptions) error {
	return nil
}

func (c *podmanClient) PruneImages(containerclient.PruneImagesOptions) (*containerclient.PruneImagesResults, error) {
	return nil, nil
}
