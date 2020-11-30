package containerclient

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/fsouza/go-dockerclient"
)

func buildTLSTransport(basePath string) (*http.Transport, error) {
	roots := x509.NewCertPool()
	pemData, err := ioutil.ReadFile(basePath + "/ca.pem")
	if err != nil {
		return nil, err
	}

	// Add the certification to the pool.
	roots.AppendCertsFromPEM(pemData)

	// Create the certificate;
	crt, err := tls.LoadX509KeyPair(basePath+"/cert.pem", basePath+"/key.pem")
	if err != nil {
		return nil, err
	}

	// Create the new tls configuration using both the authority and certificate.
	conf := &tls.Config{
		RootCAs:      roots,
		Certificates: []tls.Certificate{crt},
	}

	// Create our own transport and return it.
	return &http.Transport{
		TLSClientConfig: conf,
	}, nil
}

type dockerClient struct {
	client *docker.Client
}

func NewDockerClient(host string) (*dockerClient, error) {
	hostURL, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	// Change to an https connection if we have a cert path.
	if os.Getenv("DOCKER_CERT_PATH") != "" {
		hostURL.Scheme = "https"
	}

	c, err := docker.NewClient(hostURL.String())
	if err != nil {
		return nil, err
	}

	// Set the client to use https.
	if os.Getenv("DOCKER_CERT_PATH") != "" {
		transport, err := buildTLSTransport(os.Getenv("DOCKER_CERT_PATH"))
		if err != nil {
			return nil, err
		}

		c.HTTPClient = &http.Client{Transport: transport}
	}

	return &dockerClient{client: c}, nil
}

func (c *dockerClient) BuildImage(opts BuildImageOptions) error {
	return c.client.BuildImage(docker.BuildImageOptions{
		Name:                opts.Name,
		NoCache:             opts.NoCache,
		CacheFrom:           opts.CacheFrom,
		SuppressOutput:      opts.SuppressOutput,
		RmTmpContainer:      opts.RmTmpContainer,
		ForceRmTmpContainer: opts.ForceRmTmpContainer,
		OutputStream:        opts.OutputStream,
		RawJSONStream:       true,
		Dockerfile:          opts.Dockerfile,
		ContextDir:          opts.ContextDir,
	})
}

func (c *dockerClient) PullImage(opts PullImageOptions, auth AuthConfiguration) error {
	return c.client.PullImage(
		docker.PullImageOptions{
			Repository:    opts.Repository,
			Registry:      opts.Registry,
			Tag:           opts.Tag,
			OutputStream:  opts.OutputStream,
			RawJSONStream: true,
		},
		docker.AuthConfiguration{
			Username: auth.Username,
			Password: auth.Password,
		},
	)
}

func (c *dockerClient) PushImage(opts PushImageOptions, auth AuthConfiguration) error {
	return c.client.PushImage(
		docker.PushImageOptions{
			Name:          opts.Repository,
			Registry:      opts.Registry,
			Tag:           opts.Tag,
			OutputStream:  opts.OutputStream,
			RawJSONStream: true,
		},
		docker.AuthConfiguration{
			Username: auth.Username,
			Password: auth.Password,
		},
	)
}

func (c *dockerClient) TagImage(name string, opts TagImageOptions) error {
	return c.client.TagImage(
		name,
		docker.TagImageOptions{
			Repo:  opts.Repository,
			Tag:   opts.Tag,
			Force: true,
		},
	)
}

func (c *dockerClient) InspectImage(name string) (*Image, error) {
	dockerImage, err := c.client.InspectImage(name)
	if err != nil {
		return nil, err
	}
	return &Image{
		ID:          dockerImage.ID,
		RepoDigests: dockerImage.RepoDigests,
	}, nil
}

func (c *dockerClient) RemoveImageExtended(name string, opts RemoveImageOptions) error {
	return c.client.RemoveImageExtended(name, docker.RemoveImageOptions{Force: opts.Force})
}

func (c *dockerClient) PruneImages(opts PruneImagesOptions) (*PruneImagesResults, error) {
	pruneImageResults, err := c.client.PruneImages(docker.PruneImagesOptions{Filters: opts.Filters})
	if err != nil {
		return nil, err
	}

	imagesDeleted := []string{}
	for _, img := range pruneImageResults.ImagesDeleted {
		// We don't really care if the image was untagged or deleted
		if len(img.Untagged) > 0 {
			imagesDeleted = append(imagesDeleted, img.Untagged)
		} else {
			imagesDeleted = append(imagesDeleted, img.Deleted)
		}
	}

	return &PruneImagesResults{
		ImagesDeleted:  imagesDeleted,
		SpaceReclaimed: pruneImageResults.SpaceReclaimed,
	}, nil
}
