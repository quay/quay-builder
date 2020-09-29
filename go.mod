module github.com/quay/quay-builder

go 1.13

// https://github.com/moby/buildkit/pull/1425
replace github.com/containerd/containerd => github.com/containerd/containerd v1.3.1-0.20200227195959-4d242818bf55

replace github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200227233006-38f52c9fec82

require (
	github.com/cloudfoundry/archiver v0.0.0-20200131002800-4ca7245c29b1
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.4.2-0.20191101170500-ac7306503d23
	github.com/fsouza/go-dockerclient v1.6.5
	github.com/golang/protobuf v1.4.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/moby/buildkit v0.7.2
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/sirupsen/logrus v1.7.0
	github.com/streamrail/concurrent-map v0.0.0-20160823150647-8bf1e9bacbf6 // indirect
	github.com/ugorji/go v1.1.9 // indirect
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/beatgammit/turnpike.v2 v2.0.0-20170911161258-573f579df7ee // indirect
)
