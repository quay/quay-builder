package wamp

import (
	"testing"

	"gopkg.in/beatgammit/turnpike.v2"

	"github.com/quay/quay-builder/rpc"
)

var mappingTests = []struct {
	err         error
	expectedUri turnpike.URI
}{
	{rpc.GitCheckoutError{Err: "foobar"}, "io.quay.builder.gitcheckout"},
	{rpc.BuildPackError{Err: "foobar"}, "io.quay.builder.buildpackissue"},
	{rpc.CannotPullForCacheError{Err: "foobar"}, "io.quay.builder.cannotpullforcache"},
	{rpc.TagError{Err: "foobar"}, "io.quay.builder.tagissue"},
	{rpc.PushError{Err: "foobar"}, "io.quay.builder.pushissue"},
	{rpc.PullError{Err: "foobar"}, "io.quay.builder.cannotpullbaseimage"},
	{rpc.BuildError{Err: "foobar"}, "io.quay.builder.builderror"},
	{rpc.InvalidDockerfileError{Err: "foobar"}, "io.quay.builder.dockerfileissue"},
}

func TestErrorMapping(t *testing.T) {
	for _, mappingTest := range mappingTests {
		uri := uriFromError(mappingTest.err)
		if uri != mappingTest.expectedUri {
			t.Fatalf("Expected URI %v for %v, found: %v", mappingTest.expectedUri, uri, mappingTest.err)
		}
	}
}
