package podmanclient

import (
	"context"
	"net/url"

	"github.com/containers/podman/v2/pkg/bindings"
)

type podmanClient struct{
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
