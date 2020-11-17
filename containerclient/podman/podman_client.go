package podmanclient

import (
	"context"
	"net/url"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	"github.com/containers/podman/v2/pkg/bindings"
	"github.com/containers/podman/v2/pkg/bindings/images"
	"github.com/containers/podman/v2/pkg/domain/entities"

	"github.com/quay/quay-builder/containerclient"
)

func imagePath(repository, tag string) string {
	fullRepoPath := strings.Join([]string{repository, tag}, ":")
	return fullRepoPath
}

func fullImageRef(registry, repository, tag string) string {
	imagePath := imagePath(repository, tag)
	fullImageRef := strings.Join([]string{registry, imagePath}, "/")
	return fullImageRef
}

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

func (c *podmanClient) BuildImage(opts containerclient.BuildImageOptions) error {
	buildahOpts := imagebuildah.BuildOptions{
		NoCache:                 opts.NoCache,
		RemoveIntermediateCtrs:  opts.RmTmpContainer,
		ForceRmIntermediateCtrs: opts.ForceRmTmpContainer,
		ContextDirectory:        opts.ContextDir,
		Output:                  opts.Name,
		Quiet:                   opts.SuppressOutput,
		CommonBuildOpts:         &buildah.CommonBuildOptions{},
	}
	podmanBuildOpts := entities.BuildOptions{BuildOptions: buildahOpts}
	_, err := images.Build(c.podmanContext, []string{opts.Dockerfile}, podmanBuildOpts)
	return err
}

func (c *podmanClient) PullImage(opts containerclient.PullImageOptions, auth containerclient.AuthConfiguration) error {
	fullImagePath := fullImageRef(opts.Registry, opts.Repository, opts.Tag)
	podmanPullOpts := entities.ImagePullOptions{
		Username: auth.Username,
		Password: auth.Password,
	}
	_, err := images.Pull(c.podmanContext, fullImagePath, podmanPullOpts)
	return err
}

func (c *podmanClient) PushImage(opts containerclient.PushImageOptions, auth containerclient.AuthConfiguration) error {
	imagePath := imagePath(opts.Repository, opts.Tag)
	fullImageRef := fullImageRef(opts.Registry, opts.Repository, opts.Tag)
	podmanPushOpts := entities.ImagePushOptions{
		Username: auth.Username,
		Password: auth.Password,
	}
	err := images.Push(c.podmanContext, imagePath, fullImageRef, podmanPushOpts)
	return err
}

func (c *podmanClient) TagImage(name string, opts containerclient.TagImageOptions) error {
	err := images.Tag(c.podmanContext, name, opts.Tag, opts.Repository)
	return err
}

func (c *podmanClient) InspectImage(name string) (*containerclient.Image, error) {
	ttrue := true
	imageReport, err := images.GetImage(c.podmanContext, name, &ttrue)
	if err != nil {
		return nil, err
	}
	return &containerclient.Image{
		ID: imageReport.ImageData.ID,
		RepoDigests: imageReport.ImageData.RepoDigests,
	}, nil
}

func (c *podmanClient) RemoveImageExtended(name string, opts containerclient.RemoveImageOptions) error {
	_, err := images.Remove(c.podmanContext, name, opts.Force)
	return err
}

func (c *podmanClient) PruneImages(opts containerclient.PruneImagesOptions) (*containerclient.PruneImagesResults, error) {
	ttrue := true
	imagesDeleted, err := images.Prune(c.podmanContext, &ttrue, opts.Filters)
	if err != nil {
		return nil, err
	}
	return &containerclient.PruneImagesResults{ImagesDeleted: imagesDeleted}, nil
}
