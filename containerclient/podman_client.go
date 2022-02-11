package containerclient

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/podman/v3/pkg/bindings"
	"github.com/containers/podman/v3/pkg/bindings/images"
	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/domain/entities/reports"
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

func NewPodmanClient(host string) (*podmanClient, error) {
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

func (c *podmanClient) BuildImage(opts BuildImageOptions) error {
	buildahOpts := define.BuildOptions{
		NoCache:                 opts.NoCache,
		RemoveIntermediateCtrs:  opts.RmTmpContainer,
		ForceRmIntermediateCtrs: opts.ForceRmTmpContainer,
		ContextDirectory:        opts.ContextDir,
		Output:                  opts.Name,
		Out:                     opts.OutputStream,
		Err:                     opts.OutputStream,
		Quiet:                   opts.SuppressOutput,
		CommonBuildOpts:         &buildah.CommonBuildOptions{},
	}
	if os.Getenv("BULDAH_ISOLATION") == "chroot" {
		buildahOpts.Isolation = buildah.IsolationChroot
	}
	podmanBuildOpts := entities.BuildOptions{BuildOptions: buildahOpts}
	_, err := images.Build(c.podmanContext, []string{opts.Dockerfile}, podmanBuildOpts)
	return err
}

func (c *podmanClient) PullImage(opts PullImageOptions, auth AuthConfiguration) error {
	fullImagePath := imagePath(opts.Repository, opts.Tag)
	podmanPullOpts := images.PullOptions{
		Username: &auth.Username,
		Password: &auth.Password,
	}
	_, err := images.Pull(c.podmanContext, fullImagePath, &podmanPullOpts)
	return err
}

func (c *podmanClient) PushImage(opts PushImageOptions, auth AuthConfiguration) error {

	imagePath := imagePath(opts.Repository, opts.Tag)
	podmanPushOpts := images.PushOptions{
		Username: &auth.Username,
		Password: &auth.Password,
	}
	err := images.Push(c.podmanContext, imagePath, imagePath, &podmanPushOpts)
	return err
}

func (c *podmanClient) TagImage(name string, opts TagImageOptions) error {
	err := images.Tag(c.podmanContext, name, opts.Tag, opts.Repository, &images.TagOptions{})
	return err
}

func (c *podmanClient) InspectImage(name string) (*Image, error) {
	getOptions := images.GetOptions{}
	imageReport, err := images.GetImage(c.podmanContext, name, &getOptions)
	if err != nil {
		return nil, err
	}
	return &Image{
		ID:          imageReport.ImageData.ID,
		RepoDigests: imageReport.ImageData.RepoDigests,
	}, nil
}

func (c *podmanClient) RemoveImageExtended(name string, opts RemoveImageOptions) error {
	removeOptions := images.RemoveOptions{
		Force: &opts.Force,
	}
	_, err := images.Remove(c.podmanContext, []string{name}, &removeOptions)
	if len(err) > 0 {
		return err[0]
	}
	return nil
}

func (c *podmanClient) PruneImages(opts PruneImagesOptions) (*PruneImagesResults, error) {
	pruneOptions := images.PruneOptions{
		Filters: opts.Filters,
	}
	imagesDeletedReports, err := images.Prune(c.podmanContext, &pruneOptions)
	if err != nil {
		return nil, err
	}
	imagesDeleted := reports.PruneReportsIds(imagesDeletedReports)
	return &PruneImagesResults{ImagesDeleted: imagesDeleted}, nil
}
