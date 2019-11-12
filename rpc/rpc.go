package rpc

import (
	"errors"
	"fmt"
)

// Phase represents the milestones in progressing through a build.
type Phase string

const (
	Starting         Phase = "starting"
	Unpacking        Phase = "unpacking"
	CheckingCache    Phase = "checking-cache"
	PrimingCache     Phase = "priming-cache"
	PullingBaseImage Phase = "pulling"
	Building         Phase = "building"
	Pushing          Phase = "pushing"
)

// InvalidDockerfileError is the type of error returned from a BuildCallback when the
// provided BuildArgs do not have parsable Dockerfile.
type InvalidDockerfileError struct{ Err string }

func (e InvalidDockerfileError) Error() string {
	return e.Err
}

// BuildPackError is the type of error returned from a BuildCallback when the
// provided BuildArgs do not have sufficient data to download a BuildPack.
type BuildPackError struct{ Err string }

func (e BuildPackError) Error() string {
	return e.Err
}

// GitCheckoutError is the type of error returned from a BuildCallback when the
// provided git ref cannot be checked out.
type GitCheckoutError struct{ Err string }

func (e GitCheckoutError) Error() string {
	return e.Err
}

// GitCloneError is the type of error returned from a BuildCallback when the
// git clone fails.
type GitCloneError struct{ Err string }

func (e GitCloneError) Error() string {
	return e.Err
}

// CannotPullForCacheError is the type of error returned from a BuildCallback
// when it fails to pull the image used for caching.
type CannotPullForCacheError struct{ Err string }

func (e CannotPullForCacheError) Error() string {
	return e.Err
}

// TagError is the type of error returned from a BuildCallback
// when it fails to tag the built image.
type TagError struct{ Err string }

func (e TagError) Error() string {
	return e.Err
}

// PushError is the type of error returned from a BuildCallback
// when it fails to push the built image.
type PushError struct{ Err string }

func (e PushError) Error() string {
	return e.Err
}

// PullError is the type of error returned from a BuildCallback
// when it fails to pull the base image.
type PullError struct{ Err string }

func (e PullError) Error() string {
	return e.Err
}

// BuildError is the type of error returned from a BuildCallback
// when it fails to build the image.
type BuildError struct{ Err string }

func (e BuildError) Error() string {
	return e.Err
}

// ErrClientRejectedPhaseTransition is the type of error
// returned when buildman rejects a phase transition
type ErrClientRejectedPhaseTransition struct{ Err string }

func (e ErrClientRejectedPhaseTransition) Error() string {
	return e.Err
}

// BuildArgsBaseImage represents the arguments for a base image. The arguments
// are as follows:
//
// username - the username for pulling the base image (if any), and
// password - the password for pulling the base image (if any).
type BuildArgsBaseImage struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// BuildArgsGit represents the arguments related to git (if any). The arguments
// are as follows:
//
// url - URL to clone a repository,
// sha - commit identifier to checkout, and
// private_key - ssh private key needed to clone a repository.
type BuildArgsGit struct {
	URL        string `mapstructure:"url"`
	SHA        string `mapstructure:"sha"`
	PrivateKey string `mapstructure:"private_key"`
}

// BuildArgs represents the arguments needed to build an image. The
// arguments are as follows:
//
// build_package - URL to the build package to download and untar/unzip,
// sub_directory - location within the build package of the Dockerfile and the
//                 build context,
// dockerfile_name - name of the dockerfile within the sub_directory
// repository - repository for which this build is occurring,
// registry - registry for which this build is occuring (e.g. 'quay.io',
//            'staging.quay.io'),
// pull_token - token to use when pulling the cache for building,
// push_token - token to use to push the built image,
// tag_names - name(s) of the tag(s) for the newly built image,
// cached_tag - tag in the repository to pull to prime the cache,
// git - optional git values and credentials used to clone the repository, and
// base_image - image name and credentials used to conduct the base image pull.
type BuildArgs struct {
	BuildPackage   string             `mapstructure:"build_package"`
	Context        string             `mapstructure:"context"`
	DockerfilePath string             `mapstructure:"dockerfile_path"`
	Repository     string             `mapstructure:"repository"`
	Registry       string             `mapstructure:"registry"`
	PullToken      string             `mapstructure:"pull_token"`
	PushToken      string             `mapstructure:"push_token"`
	TagNames       []string           `mapstructure:"tag_names"`
	Git            *BuildArgsGit      `mapstructure:"git"`
	BaseImage      BuildArgsBaseImage `mapstructure:"base_image"`
}

// FullRepoName is a helper function to concatenate the registry and repository.
func (args *BuildArgs) FullRepoName() string {
	return fmt.Sprintf("%s/%s", args.Registry, args.Repository)
}

// TagMetadata is collection of a particular Docker tag's metadata.
type TagMetadata struct {
	BaseImage    string
	BaseImageTag string
	BaseImageID  string
}

// PullMetadata represents the metadata being used to pull an image when setting
// the Phase to one related to pulling.
type PullMetadata struct {
	RegistryURL  string
	BaseImage    string
	BaseImageTag string
	PullUsername string
}

// BuildMetadata is a collection of metadata about the successfully created
// build artifact.
type BuildMetadata struct {
	ImageID string
	Digests []string
}

// BuildCallback is a type of function that can be executed via remote RPC
// from a BuildManager when a build is issued.
type BuildCallback func(Client, *BuildArgs) (*BuildMetadata, error)

// ErrNoSimilarTags is returned from a Client when FindMostSimilarTag fails
// to find any similar tags.
var ErrNoSimilarTags = errors.New("failed to find any similar tags")

// Client represents an implementation of a transport between a Builder and a
// BuildManager.
type Client interface {
	// Connect establishes a connection to a BuildManager.
	Connect(endpoint string) error

	// ListenAndServe blocks awaiting a request to build from a BuildManager.
	ListenAndServe()

	// SetPhase informs a BuildManager of a transition between Phases.
	SetPhase(Phase, *PullMetadata) error

	// RegisterBuildCallback configures a client to call the provided
	// BuildCallback when a request to build is received from a BuildManager.
	RegisterBuildCallback(BuildCallback) error

	// FindMostSimilarTag sends a synchronous request to a BuildManager in order
	// to determine if there is a suitable docker tag to pull in order to prime
	// the docker build cache.
	FindMostSimilarTag(TagMetadata) (string, error)

	// PublishBuildLogEntry records a docker daemon log entry to a BuildManager.
	PublishBuildLogEntry(entry string) error
}
