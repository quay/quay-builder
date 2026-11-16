package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/quay/quay-builder/rpc"
)

type fakeClient struct {
	phases   []rpc.Phase
	phaseErr error
}

func (c *fakeClient) Ping() (bool, error)                             { return true, nil }
func (c *fakeClient) RegisterBuildJob(string) (*rpc.BuildArgs, error) { return nil, nil }
func (c *fakeClient) Heartbeat(context.Context)                       {}
func (c *fakeClient) FindMostSimilarTag(rpc.TagMetadata) (string, error) {
	return "", nil
}
func (c *fakeClient) PublishBuildLogEntry(string) error { return nil }
func (c *fakeClient) SetPhase(p rpc.Phase, _ *rpc.PullMetadata) error {
	c.phases = append(c.phases, p)
	return c.phaseErr
}

func TestReportBuildError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		phaseErr error
		want     []rpc.Phase
	}{
		{"build error", rpc.BuildError{Err: "returned a non-zero code: 1"}, nil, []rpc.Phase{rpc.Error}},
		{"other error", errors.New("pull failed"), nil, nil},
		{"set phase fails", rpc.BuildError{Err: "boom"}, errors.New("rejected"), []rpc.Phase{rpc.Error}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeClient{phaseErr: tt.phaseErr}
			reportBuildError(c, tt.err)
			if !reflect.DeepEqual(c.phases, tt.want) {
				t.Errorf("phases = %v, want %v", c.phases, tt.want)
			}
		})
	}
}
