module github.com/quay/quay-builder

go 1.16

// https://github.com/moby/buildkit/pull/1425
// replace github.com/containerd/containerd => github.com/containerd/containerd v1.3.1-0.20200227195959-4d242818bf55

//https://github.com/moby/moby/issues/40185
// replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200227233006-38f52c9fec82

// Workaround for darwin: https://github.com/ory/dockertest/issues/208
// replace golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6

require (
	code.cloudfoundry.org/archiver v0.0.0-20200131002800-4ca7245c29b1 // indirect
	github.com/cloudfoundry/archiver v0.0.0-20200131002800-4ca7245c29b1
	github.com/containerd/containerd v1.5.8 // indirect
	github.com/containers/buildah v1.23.4
	github.com/containers/common v0.44.4 // indirect
	github.com/containers/image/v5 v5.17.0 // indirect
	github.com/containers/podman/v3 v3.4.2
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.11+incompatible
	github.com/fsouza/go-dockerclient v1.7.4
	github.com/golang/protobuf v1.5.2
	github.com/moby/buildkit v0.8.3
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/sirupsen/logrus v1.8.1
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
)
