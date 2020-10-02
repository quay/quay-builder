package main

import (
	"context"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/quay/quay-builder/buildctx"
	"github.com/quay/quay-builder/rpc"
	"github.com/quay/quay-builder/rpc/grpcbuild"
	"github.com/quay/quay-builder/version"
)

const (
	connectTimeout = 10 * time.Second
)

func main() {
	// Grab the environment.
	dockerHost := os.Getenv("DOCKER_HOST")
	token := os.Getenv("TOKEN")
	endpoint := os.Getenv("ENDPOINT")
	server := os.Getenv("SERVER")
	certFile := os.Getenv("TLS_CERT_PATH")

	log.Infof("starting quay-builder: %s", version.Version)

	if endpoint == "" && server == "" {
		log.Fatal("missing or empty ENDPOINT and SERVER env vars: one is required")
	} else if endpoint == "" {
		endpoint = server + "/b1/buildmanager"
	}

	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	// Connection options
	var opts []grpc.DialOption

	// Attempt to load the TLS config.
	if len(certFile) > 0 {
		tlsCfg, err := credentials.NewClientTLSFromFile(certFile, server)
		if err != nil {
			log.Fatalf("invalid TLS config: %s", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsCfg))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// Attempt to connect to gRPC server (blocking)
	log.Infof("connecting to gRPC server...: %s", endpoint)
	opts = append(opts, grpc.WithBlock(), grpc.WithTimeout(connectTimeout))
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		log.Fatalf("failed to dial grpc server: %v", err)
	}
	defer conn.Close()

	// Create a new RPC client instance
	log.Infof("pinging buildmanager...")
	ctx, cancel := context.WithCancel(context.Background())
	rpcClient, err := grpcbuild.NewClient(ctx, conn)
	defer cancel()
	if err != nil {
		log.Fatalf("failed to connect to build manager: %s", err)
	}

	// Attempt to register the build job from TOKEN
	log.Infof("registering job for registration token: %s", token)
	buildargs, err := rpcClient.RegisterBuildJob(token)
	if err != nil {
		log.Fatalf("failed to register job to build manager: %s", err)
	}

	// Start heartbeating
	log.Infof("starting heartbeat to buildmanager")
	hbCtx, hbCancel := context.WithCancel(context.Background())
	defer hbCancel()
	go rpcClient.Heartbeat(hbCtx)

	// Start build
	log.Infof("starting build")
	_, err = build(dockerHost, rpcClient, buildargs)
	if err != nil {
		log.Fatalf("failed to build buildpack: %s", err)
	}

	log.Infof("done")
}

func build(dockerHost string, client rpc.Client, args *rpc.BuildArgs) (*rpc.BuildMetadata, error) {
	var buildCtx *buildctx.Context
	buildCtx, err := buildctx.New(client, args, dockerHost)
	if err != nil {
		return nil, err
	}

	// Unpack the buildpack.
	log.Infof("build: upacking build")
	if err = buildCtx.Unpack(); err != nil {
		return nil, err
	}

	// Pull the base image.
	log.Infof("build: pulling base image")
	if err = buildCtx.Pull(); err != nil {
		return nil, err
	}

	// Prime the cache.
	log.Infof("build: priming cache")
	if err = buildCtx.Cache(); err != nil {
		return nil, err
	}

	// Kick off the build.
	log.Infof("build: buliding")
	if err = buildCtx.Build(); err != nil {
		return nil, err
	}

	// Push the newly created image to the requested tag(s).
	log.Infof("build: pushing")
	bmd, err := buildCtx.Push()
	if err != nil {
		return nil, err
	}

	// Cleanup any pulled images.
	log.Infof("build: cleanup")
	if err = buildCtx.Cleanup(bmd.ImageID); err != nil {
		return nil, err
	}

	return bmd, nil
}
