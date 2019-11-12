package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/buildctx"
	"github.com/quay/quay-builder/rpc"
	"github.com/quay/quay-builder/rpc/wamp"
	"github.com/quay/quay-builder/version"
)

func main() {
	// Grab the environment.
	realm := os.Getenv("REALM")
	dockerHost := os.Getenv("DOCKER_HOST")
	token := os.Getenv("TOKEN")
	endpoint := os.Getenv("ENDPOINT")
	server := os.Getenv("SERVER")
	certFile := os.Getenv("TLS_CERT_PATH")
	keyFile := os.Getenv("TLS_KEY_PATH")
	caFile := os.Getenv("TLS_CA_PATH")

	// Prefix all logs with our realm.
	log.SetFormatter(newRealmFormatter(realm))

	log.Infof("starting quay-builder: %s", version.Version)

	// Check for missing env vars. Depending on the security model, REALM can be
	// obtained from the Build Manager and TOKEN may not need to exist.
	if endpoint == "" && server == "" {
		log.Fatal("missing or empty ENDPOINT and SERVER env vars: one is required")
	} else if endpoint == "" {
		endpoint = server + "/b1/socket"
	}

	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	// Attempt to load the TLS config.
	tlscfg, err := LoadTLSClientConfig(certFile, keyFile, caFile)
	if err != nil {
		log.Fatalf("invalid TLS config: %s", err)
	}

	// Create a new RPC client instance.
	rpcClient, err := wamp.NewClient(realm, token, tlscfg)
	if err != nil {
		log.Fatalf("failed to create new rpc client: %s", err)
	}

	err = rpcClient.Connect(endpoint)
	if err != nil {
		log.Fatalf("failed to connect to BuildManager: %s", err)
	}

	err = rpcClient.RegisterBuildCallback(buildCallbackClosure(dockerHost))
	if err != nil {
		log.Fatalf("failed to register build callback: %s", err)
	}

	rpcClient.ListenAndServe()
}

func buildCallbackClosure(dockerHost string) rpc.BuildCallback {
	return func(client rpc.Client, args *rpc.BuildArgs) (bmd *rpc.BuildMetadata, err error) {
		var ctx *buildctx.Context
		ctx, err = buildctx.New(client, args, dockerHost)
		if err != nil {
			return
		}

		// Unpack the buildpack.
		if err = ctx.Unpack(); err != nil {
			return
		}

		// Pull the base image.
		if err = ctx.Pull(); err != nil {
			return
		}

		// Prime the cache.
		if err = ctx.Cache(); err != nil {
			return
		}

		// Kick off the build.
		if err = ctx.Build(); err != nil {
			return
		}

		// Push the newly created image to the requested tag(s).
		bmd, err = ctx.Push()
		if err != nil {
			return
		}

		// Cleanup any pulled images.
		if err = ctx.Cleanup(bmd.ImageID); err != nil {
			return
		}

		return
	}
}
