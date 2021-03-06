package main

import (
	"context"
	"crypto/tls"
	"os"
	"strings"
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
	containerRuntime := os.Getenv("CONTAINER_RUNTIME")
	dockerHost := os.Getenv("DOCKER_HOST")
	token := os.Getenv("TOKEN")
	server := os.Getenv("SERVER")
	certFile := os.Getenv("TLS_CERT_PATH")
	insecure := os.Getenv("INSECURE")

	log.Infof("starting quay-builder: %s", version.Version)

	if server == "" {
		log.Fatal("missing or empty SERVER env vars: required format <host>:<port>")
	}

	if containerRuntime == "" {
		containerRuntime = "docker"
	}

	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	// Connection options
	var opts []grpc.DialOption

	// Attempt to load the TLS config.
	if len(certFile) > 0 {
		tlsCfg, err := credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			log.Fatalf("invalid TLS config: %s", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(tlsCfg))
	} else if strings.ToLower(insecure) == "true" {
		opts = append(opts, grpc.WithInsecure())
	} else {
		// Load the default system certs
		tlsCfg := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(tlsCfg))
	}

	// Attempt to connect to gRPC server (blocking)
	log.Infof("connecting to gRPC server...: %s", server)
	opts = append(opts, grpc.WithBlock(), grpc.WithTimeout(connectTimeout))
	conn, err := grpc.Dial(server, opts...)
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
	_, err = build(dockerHost, containerRuntime, rpcClient, buildargs, hbCancel)
	if err != nil {
		log.Fatalf("failed to build buildpack: %s", err)
	}

	log.Infof("done")
}

func build(dockerHost, containerRuntime string, client rpc.Client, args *rpc.BuildArgs, hbCanceller context.CancelFunc) (*rpc.BuildMetadata, error) {
	var buildCtx *buildctx.Context
	buildCtx, err := buildctx.New(client, args, dockerHost, containerRuntime)
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
	log.Infof("build: building")
	if err = buildCtx.Build(); err != nil {
		return nil, err
	}

	// Push the newly created image to the requested tag(s).
	log.Infof("build: pushing")
	bmd, err := buildCtx.Push()
	if err != nil {
		return nil, err
	}

	// Stop heartbeats
	hbCanceller()

	// Move build to completed phase
	if err := client.SetPhase(rpc.Complete, nil); err != nil {
		log.Errorf("failed to update phase to `complete`")
		return nil, err
	}

	// Cleanup any pulled images.
	log.Infof("build: cleanup")
	if err = buildCtx.Cleanup(bmd.ImageID); err != nil {
		return nil, err
	}

	return bmd, nil
}
