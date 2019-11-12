package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

const serverName = "quay-services"

// LoadTLSClientConfig initializes a *tls.Config using the given certificates
// and private key, that can be used to communicate with a server using client
// certificate authentication.
//
// If no certificates are given, a nil *tls.Config is returned.
// The CA certificate is optional and falls back to the system default.
func LoadTLSClientConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	var caCertPool *x509.CertPool
	if caFile != "" {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
	}

	tlsConfig := &tls.Config{
		ServerName:   serverName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	return tlsConfig, nil
}
