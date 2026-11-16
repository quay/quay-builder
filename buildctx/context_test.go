package buildctx

import (
	"context"
	"errors"
	"testing"

	"github.com/quay/quay-builder/containerclient"
	"github.com/quay/quay-builder/rpc"
)

type fakeRPC struct{ phases []rpc.Phase }

func (c *fakeRPC) Ping() (bool, error)                                { return true, nil }
func (c *fakeRPC) RegisterBuildJob(string) (*rpc.BuildArgs, error)    { return nil, nil }
func (c *fakeRPC) Heartbeat(context.Context)                          {}
func (c *fakeRPC) FindMostSimilarTag(rpc.TagMetadata) (string, error) { return "", nil }
func (c *fakeRPC) PublishBuildLogEntry(string) error                  { return nil }
func (c *fakeRPC) SetPhase(p rpc.Phase, _ *rpc.PullMetadata) error {
	c.phases = append(c.phases, p)
	return nil
}

// failingBuild embeds the interface so only BuildImage needs an implementation.
type failingBuild struct{ containerclient.Client }

func (failingBuild) BuildImage(containerclient.BuildImageOptions) error {
	return errors.New("The command '/bin/sh -c exit 1' returned a non-zero code: 1")
}

func TestBuildReturnsBuildError(t *testing.T) {
	r := &fakeRPC{}
	bc := &Context{
		client:          r,
		writer:          containerclient.NewRPCWriter(r, "docker"),
		containerClient: failingBuild{},
		args:            &rpc.BuildArgs{Registry: "quay.io", Repository: "ns/repo"},
		buildpackDir:    t.TempDir(),
	}

	err := bc.Build()
	if !errors.As(err, &rpc.BuildError{}) {
		t.Fatalf("Build() error = %#v, want rpc.BuildError", err)
	}
	if len(r.phases) != 1 || r.phases[0] != rpc.Building {
		t.Errorf("phases = %v, want [%s]", r.phases, rpc.Building)
	}
}
