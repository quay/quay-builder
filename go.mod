module github.com/quay/quay-builder

go 1.12

require (
	code.cloudfoundry.org/archiver v0.0.0-20180525162158-e135af3d5a2a // indirect
	github.com/cloudfoundry/archiver v0.0.0-20180525162158-e135af3d5a2a
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
	github.com/docker/docker v0.7.3-0.20190212235812-0111ee70874a
	github.com/fsouza/go-dockerclient v1.3.6
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/mitchellh/mapstructure v0.0.0-20150528213339-482a9fd5fa83
	github.com/moby/buildkit v0.4.0
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/sirupsen/logrus v1.3.0
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/streamrail/concurrent-map v0.0.0-20160823150647-8bf1e9bacbf6 // indirect
	github.com/ugorji/go/codec v0.0.0-20190320090025-2dc34c0b8780 // indirect
	gopkg.in/beatgammit/turnpike.v2 v2.0.0-20170911161258-573f579df7ee
)

replace github.com/containerd/containerd v1.3.0-0.20190212172151-f5b0fa220df8 => github.com/containerd/containerd v1.3.0
