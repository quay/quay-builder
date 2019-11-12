package docker

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/fsouza/go-dockerclient"
)

// Client is an interface for all of the Docker™ interactions required of a
// worker.
type Client interface {
	BuildImage(docker.BuildImageOptions) error
	PullImage(docker.PullImageOptions, docker.AuthConfiguration) error
	PushImage(docker.PushImageOptions, docker.AuthConfiguration) error
	TagImage(string, docker.TagImageOptions) error
	InspectImage(string) (*docker.Image, error)
	RemoveImageExtended(string, docker.RemoveImageOptions) error
	PruneImages(docker.PruneImagesOptions) (*docker.PruneImagesResults, error)
}

// NewClient returns a new connection to the Docker daemon running at the
// provided host.
func NewClient(host string) (*docker.Client, error) {
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

	return c, nil
}

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
