# Quay Builder

This repository is for an automated build worker for a Quay.

## Architecture

There is a client/server relationship between builder and the management server.
Clients connect using a standard websocket RPC/pubsub subprotocol called [WAMP](http://wamp.ws).
There are two modes in which builders can operate: enterprise and hosted.
Enterprise builders are designed to be long-running processes on the given machine that will be trusted forever.
In this mode a builder connect to a Build Manager and indefinitely loop completing available work.
Hosted builders are designed to be dynamically created and connect to the management server for a single build and then disappear.

## Building the builder

```
make test
make build
```

## Running the builder

### Enterprise

Only an endpoint is required as all other parameters for building are acquired from a Build Manager on a per build basis.

```sh
ENDPOINT="ws://localhost:8787" ./quay-builder
```

### Hosted

A token and realm must be provided at launch in order to identify a particular build or else it will be rejected by a Build Manager.

```sh
TOKEN="sometoken" ENDPOINT="ws://localhost:8787" REALM="builder-realm" ./quay-builder
```

## Building the builder image

For both images, you can also specify make parameters

`IMAGE_TAG` ( tag name, default to `latest`) 

`IMAGE` ( repo name, default to `quay.io/quay/quay-builder`) 

and the built image will be tagged with 
```
<IMAGE>:<IMAGE_TAG>-<base image name>
```
where the `<base image name>` can be either `alpine` or `rhel7`.

### Building Alpine based image:
```sh
make build-alpine-image
```
This generates image with tag `quay.io/quay/quay-builder:latest-alpine`.

### Building RHEL based image 
It requires certificate key and requires enabling `--squash` experimental feature
```sh
make build-rhel7-image SUBSCRIPTION_KEY=<path to your key file (PEM)>
```
This generates image with tag `quay.io/quay/quay-builder:latest-rhel7`.

## Running the builder image

Running alpine based image or rhel based image requires the same parameters but different image.

**Please Notice** that quay builder uses the host machine's docker.sock to pull/push/build images and therefore, the docker machine must be able to reach the Quay host. You can debug by pushing a image to quay instance.

### Pointing to Quay without TLS
```
docker run --restart on-failure -e SERVER=ws://myquayserver:8787 -v /var/run/docker.sock:/var/run/docker.sock quay.io/quay/quay-builder:latest-alpine
```

### Pointing to Quay with TLS
```
docker run --restart on-failure -e SERVER=wss://myquayserver:8787 -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/customCA/rootCA.pem:/certs/rootCA.pem quay.io/quay/quay-builder:latest-alpine
```
