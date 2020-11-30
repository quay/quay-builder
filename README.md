# Quay Builder

This repository is for an automated build worker for a Quay.

## Architecture

There is a client/server relationship between builder and the management server.
Builders are created and connect to the build manager using the GRPC protocol.
Builders are designed to be dynamically created and connect to the management server for a single build and then disappear,
generally on some control plane such as K8s or AWS.

## Building the builder

```
make test
make build
```

## Running the builder

### Environment variables

The builders are bootstrapped and configured using environment variables. These are set when created by the build manager.
The parameters necessary for the actual build are obtained in a subsequent call to the build manager's API

`CONTAINER_RUNTIME`: "podman" or "docker"
`DOCKER_HOST`: The container runtime socket. Defaults to "unix:///var/run/docker.sock"
`TOKEN`: The registration token needed to get the build args from the build manager
`SERVER`: The build manager's GRPC endpoint. Format: <host>:<port>
`TLS_CERT_PATH`: TLS cert file path (optional)
`INSECURE`: "true" or "false". Of "true" attempt to connect to the build manager without tls.

### Container runtimes

The builder supports Docker and Podman/Buildah to run the builds. The runtime is specified using the `CONTAINER_RUNTIME` and `DOCKER_HOST`.
If these ENV variables are not set, `CONTAINER_RUNTIME` and `DOCKER_HOST` will be set to "docker" and "unix:///var/run/docker.sock", respectively.
If `CONTAINER_RUNTIME` is set to "podman", it is expected that `DOCKER_HOST` is set to podman's equivalent to the docker's docker. e.g unix:///var/run/podman.sock

## Building the builder image

For both images, you can also specify make parameters

`IMAGE_TAG` ( tag name, default to `latest`) 

`IMAGE` ( repo name, default to `quay.io/quay/quay-builder`) 

and the built image will be tagged with 
```
<IMAGE>:<IMAGE_TAG>-<base image name>
```
where the `<base image name>` can be either `alpine` or `centos`.

### Building Alpine based image:
```sh
make build-alpine
```
This generates image with tag `quay.io/projectquay/quay-builder:latest-alpine`.

### Building CentOS based image:
```sh
make build-centos
```
This generates image with tag `quay.io/projectquay/quay-builder:latest-centos`.
