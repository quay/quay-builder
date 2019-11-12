package buildctx

import (
	"fmt"
	"os"
	"strings"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/nu7hatch/gouuid"
	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/buildpack"
	"github.com/quay/quay-builder/docker"
	"github.com/quay/quay-builder/docker/dockerfile"
	"github.com/quay/quay-builder/rpc"
)

// scratch is a special case, empty base image. It is not listed after
// executing `docker images`, but `docker pull scratch` will pull the image
// revealing a short ID of "511136ea3c5a" which can be `docker inspect`ed
// to reveal the full image ID.
const (
	scratchImageName = "scratch"
	scratchImageID   = "511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158"
)

// Context represents the internal state of a build.
type Context struct {
	client       rpc.Client
	writer       docker.LogWriter
	dockerClient docker.Client
	args         *rpc.BuildArgs
	metadata     *dockerfile.Metadata
	buildpackDir string
	buildID      string
	cacheTag     string
}

// New connects to the docker daemon and sets up the initial state of a build
// context.
//
// If the connection to the docker daemon fails, exits with log.Fatal.
func New(client rpc.Client, args *rpc.BuildArgs, dockerHost string) (*Context, error) {
	// Connect to the local docker client.
	log.Infof("connecting to docker host: %s", dockerHost)
	dockerClient, err := docker.NewClient(dockerHost)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("connected to docker host: %s", dockerHost)

	return &Context{
		client:       client,
		writer:       docker.NewRPCWriter(client),
		dockerClient: dockerClient,
		args:         args,
	}, nil
}

// Unpack downloads and expands the buildpack and parses the Dockerfile.
func (bc *Context) Unpack() error {
	if err := bc.client.SetPhase(rpc.Unpacking, nil); err != nil {
		return err
	}

	// Download and expand the buildpack.
	buildpackDir, err := buildpack.Download(bc.args)
	if err != nil {
		return err
	}
	bc.buildpackDir = buildpackDir

	// Parse the Dockerfile.
	metadata, err := dockerfile.NewMetadataFromDir(buildpackDir, bc.args.DockerfilePath)

	if err != nil {
		return err
	}
	bc.metadata = metadata

	return nil
}

// Pull executes "docker pull" for the base image of the build's Dockerfile.
func (bc *Context) Pull() error {
	if err := bc.client.SetPhase(rpc.PullingBaseImage, &rpc.PullMetadata{
		RegistryURL:  bc.args.Registry,
		BaseImage:    bc.metadata.BaseImage,
		BaseImageTag: bc.metadata.BaseImageTag,
		PullUsername: bc.args.BaseImage.Username,
	}); err != nil {
		return err
	}

	return pullBaseImage(bc.writer, bc.dockerClient, bc.metadata, bc.args)
}

// Cache calls an RPC to the BuildManager to find the best tag to pull for
// caching and then "docker pull"s it.
func (bc *Context) Cache() error {
	if err := bc.client.SetPhase(rpc.CheckingCache, nil); err != nil {
		return err
	}

	// Attempt to calculate the optimal tag. If we cannot find a tag, then caching is simply
	// skipped.
	cachedTag, err := findCachedTag(bc.writer, bc.client, bc.dockerClient, bc.metadata)
	if err != nil {
		log.Warningf("Failed to lookup caching tag: %v", err)
		return nil
	}

	// Conduct a pull of the existing tag (if any). This will prime the cache.
	if bc.args.PullToken != "" && cachedTag != "" {
		if err := bc.client.SetPhase(rpc.PrimingCache, nil); err != nil {
			return err
		}

		err = primeCache(bc.writer, bc.dockerClient, bc.args, cachedTag)
		if err != nil {
			log.Warningf("Error priming cache: %s", err.Error())
		} else {
			bc.cacheTag = cachedTag
		}
	}

	return nil
}

// Build performs a "docker build".
func (bc *Context) Build() error {
	if err := bc.client.SetPhase(rpc.Building, nil); err != nil {
		return err
	}

	// Clean up the buildpack.
	defer func() {
		if err := os.RemoveAll(bc.buildpackDir); err != nil {
			log.Errorf("failed to remove buildpack from filesystem: %s", err)
		} else {
			log.Infof("removed build dir: %s", bc.buildpackDir)
		}
	}()
	var err error
	bc.buildID, err = executeBuild(bc.writer, bc.dockerClient, bc.buildpackDir,
		bc.args.DockerfilePath, bc.args.FullRepoName(), bc.cacheTag)
	return err
}

// Push executes "docker push" and builds a successful call result if no
// failures occur.
func (bc *Context) Push() (*rpc.BuildMetadata, error) {
	if err := bc.client.SetPhase(rpc.Pushing, nil); err != nil {
		return nil, err
	}

	imageID, digests, err := pushBuiltImage(bc.writer, bc.dockerClient, bc.args, bc.buildID)
	if err != nil {
		return nil, err
	}

	return &rpc.BuildMetadata{ImageID: imageID, Digests: digests}, nil
}

// retryDockerRequest retries attempts to execute a closure that alters that
// state of the docker daemon until it succeeds.
func retryDockerRequest(w docker.LogWriter, requestFunc func() error) (err error) {
	for i := 0; i < 3; i++ {
		// Explicitly throw away the errors from any previous attempts to pull.
		w.ResetError()

		err = requestFunc()
		rerr, hasResponseError := w.ErrResponse()
		if err == nil && !hasResponseError {
			return nil
		}

		log.Infof("failed docker request attempt #%d: err: %s err response %s", i, err, rerr)
		if i == 2 {
			if err != nil {
				return err
			}

			return rerr
		}
	}

	return nil
}

func primeCache(w docker.LogWriter, dockerClient docker.Client, args *rpc.BuildArgs, cachedTag string) error {
	if cachedTag == "" {
		// There's nothing to do!
		return nil
	}

	log.Infof("priming cache with image %s:%s", args.Repository, cachedTag)

	// Attempt to pull the existing tag (if any) three times.
	err := retryDockerRequest(w, func() error {
		return dockerClient.PullImage(
			dockerclient.PullImageOptions{
				Repository:    args.FullRepoName(),
				Registry:      args.Registry,
				Tag:           cachedTag,
				OutputStream:  w,
				RawJSONStream: true,
			},
			dockerclient.AuthConfiguration{
				Username: "$token",
				Password: args.PullToken,
			},
		)
	})
	if err != nil {
		return rpc.CannotPullForCacheError{Err: err.Error()}
	}

	return nil
}

func pullBaseImage(w docker.LogWriter, dockerClient docker.Client, df *dockerfile.Metadata, args *rpc.BuildArgs) error {
	// Skip pulling the base image if it's "scratch" which is a built-in image
	// that throws an error after executing `docker pull`.
	if df.BaseImage == scratchImageName {
		return nil
	}

	pullOptions := dockerclient.PullImageOptions{
		Registry:      args.Registry,
		Repository:    df.BaseImage,
		Tag:           df.BaseImageTag,
		OutputStream:  w,
		RawJSONStream: true,
	}

	// Only pull the base image with auth when it is in our own registry.
	var pullAuth dockerclient.AuthConfiguration
	var usesAuth bool
	if args.BaseImage.Username != "" && strings.Index(df.BaseImage, args.Registry) == 0 {
		pullAuth = dockerclient.AuthConfiguration{
			Username: args.BaseImage.Username,
			Password: args.BaseImage.Password,
		}
		usesAuth = true
	}

	log.Infof("pulling base image %s:%s (with auth: %t)", df.BaseImage, df.BaseImageTag, usesAuth)

	// Attempt to pull an image three times.
	err := retryDockerRequest(w, func() error {
		return dockerClient.PullImage(pullOptions, pullAuth)
	})
	if err != nil {
		return rpc.PullError{Err: err.Error()}
	}

	return nil
}

func findCachedTag(w docker.LogWriter, client rpc.Client, dockerClient docker.Client, df *dockerfile.Metadata) (string, error) {
	log.Infof("querying Docker for the ID of the pulled base image: %s:%s", df.BaseImage, df.BaseImageTag)
	var baseImageID string
	if df.BaseImage == scratchImageName {
		// scratch is a builtin image that must be manually assigned its proper ID.
		baseImageID = scratchImageID
	} else {
		baseImage, err := dockerClient.InspectImage(df.BaseImage + ":" + df.BaseImageTag)
		if err != nil {
			// TODO(jzelinskie): maybe make this non-fatal
			return "", err
		}

		if rerr, hasResponseError := w.ErrResponse(); hasResponseError {
			// TODO(jzelinskie): maybe make this non-fatal
			return "", rerr
		}

		baseImageID = baseImage.ID
	}

	log.Infof("querying BuildManager for most similar tag")
	return client.FindMostSimilarTag(rpc.TagMetadata{
		BaseImage:    df.BaseImage,
		BaseImageTag: df.BaseImageTag,
		BaseImageID:  baseImageID,
	})
}

func pushBuiltImage(w docker.LogWriter, dockerClient docker.Client, args *rpc.BuildArgs, imageID string) (string, []string, error) {
	// Push each new tag for the image.
	for _, tagName := range args.TagNames {
		// Setup tag options.
		tagOptions := dockerclient.TagImageOptions{
			Repo:  args.FullRepoName(),
			Tag:   tagName,
			Force: true,
		}

		// Tag the image.
		log.Infof("tagging image %s as %s:%s", imageID, args.FullRepoName(), tagName)
		err := dockerClient.TagImage(imageID, tagOptions)
		if err != nil {
			return "", nil, rpc.TagError{Err: err.Error()}
		}

		if rerr, hasResponseError := w.ErrResponse(); hasResponseError {
			return "", nil, rpc.TagError{Err: rerr.Error()}
		}

		fullyQualifiedName := args.FullRepoName() + ":" + tagName
		log.Infof("pushing image %s (%s)", fullyQualifiedName, imageID)
		err = retryDockerRequest(w, func() error {
			return dockerClient.PushImage(
				dockerclient.PushImageOptions{
					Name:          args.FullRepoName(),
					Registry:      args.Registry,
					Tag:           tagName,
					OutputStream:  w,
					RawJSONStream: true,
				},
				dockerclient.AuthConfiguration{
					Username: "$token",
					Password: args.PushToken,
				},
			)
		})
		if err != nil {
			return "", nil, rpc.PushError{Err: err.Error()}
		}

		log.Infof("successfully pushed %s", fullyQualifiedName)
	}

	// Find the image built.
	dockerImage, err := dockerClient.InspectImage(imageID)
	if err != nil {
		return "", nil, rpc.TagError{Err: err.Error()}
	}

	if rerr, hasResponseError := w.ErrResponse(); hasResponseError {
		return "", nil, rpc.TagError{Err: rerr.Error()}
	}

	return dockerImage.ID, dockerImage.RepoDigests, nil
}

// Cleanup attempts to remove all the images associated with the build.
func (bc *Context) Cleanup(builtImageID string) error {
	// Remove the cached image (if any).
	if bc.cacheTag != "" {
		cacheImage := fmt.Sprintf("%s:%s", bc.args.Repository, bc.cacheTag)
		err := bc.dockerClient.RemoveImageExtended(cacheImage, dockerclient.RemoveImageOptions{
			Force: true,
		})
		if err != nil {
			log.Warningf("Could not remove cached image %s: %v", cacheImage, err)
		}
	}

	// Remove the base image.
	baseImage := bc.metadata.BaseImage
	if bc.metadata.BaseImageTag != "" {
		baseImage = fmt.Sprintf("%s:%s", baseImage, bc.metadata.BaseImageTag)
	}
	err := bc.dockerClient.RemoveImageExtended(baseImage, dockerclient.RemoveImageOptions{
		Force: true,
	})
	if err != nil {
		log.Warningf("Could not remove base image %s: %v", baseImage, err)
	}

	// Remove the built image.
	brerr := bc.dockerClient.RemoveImageExtended(builtImageID, dockerclient.RemoveImageOptions{
		Force: true,
	})
	if brerr != nil {
		log.Warningf("Could not remove built image %s: %v", builtImageID, brerr)
	}

	// Prune any other images.
	_, perr := bc.dockerClient.PruneImages(dockerclient.PruneImagesOptions{})
	if perr != nil {
		log.Warningf("Could not prune images: %v", perr)
	}

	return nil
}

func executeBuild(w docker.LogWriter, dockerClient docker.Client, buildPackageDirectory string, dockerFileName string, repo string, cacheTag string) (string, error) {
	buildUUID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	buildID := buildUUID.String()

	log.Infof("executing build with ID %s", buildID)

	cacheFrom := []string{}
	if cacheTag != "" {
		cachedImage := repo + ":" + cacheTag
		cacheFrom = []string{cachedImage}
		log.Infof("using cache image %s", cachedImage)
	}

	err = dockerClient.BuildImage(dockerclient.BuildImageOptions{
		Name:                buildID,
		NoCache:             false,
		CacheFrom:           cacheFrom,
		SuppressOutput:      false,
		RmTmpContainer:      true,
		ForceRmTmpContainer: true,
		OutputStream:        w,
		RawJSONStream:       true,
		Dockerfile:          dockerFileName, // Required for .dockerignore to work
		ContextDir:          buildPackageDirectory,
	})
	if err != nil {
		return "", rpc.BuildError{Err: err.Error()}
	}

	if rerr, hasResponseError := w.ErrResponse(); hasResponseError {
		return "", rpc.BuildError{Err: rerr.Error()}
	}

	return buildID, nil
}
