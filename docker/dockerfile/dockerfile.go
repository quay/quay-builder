package dockerfile

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/docker/distribution/reference"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"

	"github.com/quay/quay-builder/rpc"
)

var (
	// ErrInvalidBuildContext is returned when there is a failure finding the
	// location of the source files of an image.
	ErrInvalidBuildContext = rpc.InvalidDockerfileError{Err: "Invalid build pack. Are you sure it is a Dockerfile, .tar.gz or .zip archive?"}

	// ErrMissingDockerfile is returned when no Dockerfile was found.
	ErrMissingDockerfile = rpc.InvalidDockerfileError{Err: "Missing Dockerfile"}

	// ErrEmptyDockerfile is returned when a parsed Dockerfile has no nodes.
	ErrEmptyDockerfile = rpc.InvalidDockerfileError{Err: "Empty Dockerfile"}

	// ErrInvalidDockerfile is returned when a Dockerfile cannot be parsed.
	ErrInvalidDockerfile = rpc.InvalidDockerfileError{Err: "Could not parse Dockerfile"}

	// ErrDockerfileMissingFROMorARG is returned the first directive of a Dockerfile
	// isn't FROM or ARG.
	ErrDockerfileMissingFROMorARG = rpc.InvalidDockerfileError{Err: "First line in Dockerfile isn't FROM or ARG"}

	// ErrInvalidBaseImage is returned when the base image referenced in the FROM
	// directive is invalid.
	ErrInvalidBaseImage = rpc.InvalidDockerfileError{Err: "FROM line specifies an invalid base image"}
)

// Metadata represents a parsed Dockerfile.
type Metadata struct {
	BaseImage    string
	BaseImageTag string
}

// NewMetadataFromReader parses a Dockerfile reader generates metadata based on
// the contents.
func NewMetadataFromReader(r io.Reader, buildContextDirectory string) (*Metadata, error) {
	var imageAndTag string
	substitutionArgs := []string{}

	// Parse the Dockerfile.
	parsed, err := parser.Parse(bufio.NewReader(r))
	if err != nil {
		log.Errorf("Could not parse Dockerfile: %v", err)
		if strings.Contains(err.Error(), "file with no instructions") {
			return nil, ErrEmptyDockerfile
		}
		return nil, ErrInvalidDockerfile
	}

	ast := parsed.AST
	if len(ast.Children) == 0 {
		return nil, ErrEmptyDockerfile
	}

	// Make sure the first command is either FROM or ARG
	if ast.Children[0].Value != "arg" && ast.Children[0].Value != "from" {
		return nil, ErrDockerfileMissingFROMorARG
	}

	stages, metaArgs, _ := instructions.Parse(ast)
	if ast.Children[0].Value == "arg" {
		for _, metaArg := range metaArgs {
			if metaArg.Value != nil {
				substitutionArgs = append(substitutionArgs, metaArg.Key+"="+*metaArg.Value)
			}
		}
		shlex := shell.NewLex(parsed.EscapeToken)
		imageAndTag, _ = shlex.ProcessWord(stages[0].BaseName, substitutionArgs)

	} else if ast.Children[0].Value == "from" {
		imageAndTag = stages[0].BaseName
	}

	ref, err := reference.Parse(imageAndTag)
	if err != nil {
		return nil, ErrInvalidBaseImage
	}

	// Parse the image name.
	var image string
	named, ok := ref.(reference.Named)
	if !ok {
		return nil, ErrInvalidBaseImage
	}
	image = named.Name()

	// Attempt to parse the tag name.
	var tag string
	nametag, ok := ref.(reference.NamedTagged)
	if ok {
		tag = nametag.Tag()
	}
	if tag == "" {
		tag = "latest"
	}

	return &Metadata{
		BaseImage:    image,
		BaseImageTag: tag,
	}, nil
}

// NewMetadataFromDir parses a Dockerfile located within the provided directory
// and generates metadata based on the contents.
func NewMetadataFromDir(buildContextDirectory, dockerfileName string) (*Metadata, error) {
	// Load the contents of the Dockerfile.
	file, err := os.Open(path.Join(buildContextDirectory, dockerfileName))
	if err != nil {
		return nil, ErrMissingDockerfile
	}
	defer file.Close()

	return NewMetadataFromReader(file, buildContextDirectory)
}
