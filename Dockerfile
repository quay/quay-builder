FROM registry.access.redhat.com/ubi8/go-toolset:1.25.3-1765294123 as build
USER root
RUN dnf install -y --setopt=tsflags=nodocs git
COPY . /go/src/
RUN cd /go/src/ && make build

FROM registry.access.redhat.com/ubi8/podman
LABEL maintainer "Quay devel<quay-devel@redhat.com>"

RUN set -ex\
	; dnf install -y --setopt=tsflags=nodocs --setopt=skip_missing_names_on_install=False git wget \
	; dnf -y -q clean all

COPY --from=build /go/src/bin/quay-builder /usr/local/bin
COPY buildpack/ssh-git.sh /
COPY entrypoint.sh /home/podman/entrypoint.sh

# Rootless/unprivileged buildah configurations
# https://github.com/containers/buildah/blob/main/docs/tutorials/05-openshift-rootless-build.md
RUN touch /etc/subgid /etc/subuid && \
    chmod g=u /etc/subgid /etc/subuid /etc/passwd && \
    echo 'podman:100000:65536' > /etc/subuid && echo 'podman:100000:65536' > /etc/subgid && \
	# Set driver to VFS, which doesn't require host modifications compared to overlay
	# Set shortname aliasing to permissive - https://www.redhat.com/sysadmin/container-image-short-names
	mkdir -p /home/podman/.config/containers && \
    (echo '[storage]';echo 'driver = "vfs"') > /home/podman/.config/containers/storage.conf && \ 
    sed -i 's/short-name-mode="enforcing"/short-name-mode="permissive"/g' /etc/containers/registries.conf && \
	mkdir /certs /home/podman/.config/cni && chown podman:podman /certs /home/podman/.config/cni

VOLUME [ "/certs" ]

WORKDIR /home/podman

USER podman

ENTRYPOINT ["sh", "/home/podman/entrypoint.sh"]
