# syntax=docker/dockerfile:1.2
# Stream swaps to CentOS Stream once and is reused.
# FROM quay.io/centos/centos:stream8 AS stream
FROM registry.access.redhat.com/ubi8/ubi:latest as base
RUN set -ex\
	; dnf install -y gpgme-devel \
	; dnf -y -q clean all

RUN dnf install -y --setopt=tsflags=nodocs --setopt=skip_missing_names_on_install=False git perl wget make gcc


FROM base AS build

ARG BUILDER_SRC

ENV GO_VERSION=1.16.13
ENV GO_OS=linux
ENV GO_ARCH=amd64
ENV GOPATH=/go

RUN curl https://dl.google.com/go/go${GO_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz --output go.tar.gz
RUN tar -C /usr/local -xzf go.tar.gz > /dev/null
ENV PATH=$PATH:/usr/local/go/bin:${GOPATH}/bin

RUN go version

COPY . /go/src/${BUILDER_SRC}
RUN cd /go/src/${BUILDER_SRC} && go mod vendor && make build


FROM base AS final
LABEL maintainer "Quay devel<quay-devel@redhat.com>"

ARG BUILDER_SRC

COPY --from=build /go/src/${BUILDER_SRC}/bin/quay-builder /usr/local/bin

COPY buildpack/ssh-git.sh /
ADD load_extra_ca.rhel.sh /load_extra_ca.sh
ADD entrypoint.sh /entrypoint.sh

VOLUME [ "/certs" ]

ENTRYPOINT ["sh", "/entrypoint.sh"]
