package buildpack

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/archiver/extractor"
	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/rpc"
)

const (
	quayDocsSubmoduleURL = "http://docs.quay.io/guides/git-submodules.html"
	processIdleTimeout   = time.Minute
	processTimeout       = time.Minute * 15
)

// Download downloads the build package found at the given URL, returning the
// path to a temporary directory on the file system with those contents,
// extracted if necessary.
func Download(args *rpc.BuildArgs) (string, error) {
	var buildPackDir string

	switch {
	// Clone the git repository.
	case args.Git != nil:
		log.Infof("cloning buildpack: %s at %s", args.Git.SHA, args.Git.URL)
		repoDir, err := Clone(args.Git.URL, args.Git.SHA, args.Git.PrivateKey)
		if err != nil {
			return "", err
		}
		buildPackDir = repoDir

	// Download the buildpack.
	case args.BuildPackage != "":
		log.Infof("downloading buildpack: %s", args.BuildPackage)
		bpDir, err := download(args.BuildPackage)
		if err != nil {
			return "", err
		}
		buildPackDir = bpDir

	// This should never happen!
	default:
		log.Errorf("insufficient buildpack args: %#v", args)
		return "", rpc.BuildPackError{Err: "insufficient buildpack args"}
	}

	return filepath.Join(buildPackDir, args.Context), nil
}

// download downloads (and potentially extracts) non-git buildpacks.
func download(url string) (string, error) {
	// Load the build package from the URL.
	resp, err := http.Get(url)
	if err != nil {
		return "", rpc.BuildPackError{Err: err.Error()}
	}
	defer resp.Body.Close()

	// Find the MIME type of the build package and use it to untar/unzip.
	mimetype := resp.Header.Get("Content-Type")

	r := resp.Body
	if mimetype == "" {
		br := bufio.NewReaderSize(resp.Body, 4096)
		r = struct {
			*bufio.Reader
			io.Closer
		}{
			Reader: br,
			Closer: resp.Body,
		}
		const detectContentLen = 512
		b, _ := br.Peek(detectContentLen)
		mimetype = http.DetectContentType(b)
	}

	// Remove any extra data found on the mimetype (i.e. charset).
	mimetype = strings.Split(mimetype, ";")[0]

	// Extract the build package into a temp folder.
	return extractBuildPackage(r, mimetype)
}

// extractToTempDir extracts a body into a temporary directory and returns the path.
func extractToTempDir(body io.Reader, xtractor extractor.Extractor) (string, error) {
	// Create a temporary file.
	archiveFile, err := ioutil.TempFile("", "build_archive")
	if err != nil {
		return "", err
	}
	defer archiveFile.Close()

	// Copy the build archive to a temporary file (this forces the actual
	// downloading of the file if body is http.Request.Body).
	_, err = io.Copy(archiveFile, body)
	if err != nil {
		return "", err
	}

	// Create a temporary directory for the build pack.
	tempDir, err := ioutil.TempDir("", "build_pack")
	if err != nil {
		return "", err
	}

	// Extract the contents of the archive into the temporary directory.
	err = xtractor.Extract(archiveFile.Name(), tempDir)
	if err != nil {
		return "", err
	}

	return tempDir, nil
}

// dockerfileTempDir creates a temporary directory, copying over the dockerfile.
func dockerfileTempDir(dockerfile io.Reader) (string, error) {
	// Create a directory containing the Dockerfile directly.
	tempDir, err := ioutil.TempDir("", "build_pack")
	if err != nil {
		return "", err
	}

	// Create the Dockerfile inside the directory.
	fo, err := os.Create(tempDir + "/Dockerfile")
	if err != nil {
		return "", err
	}
	defer fo.Close()

	// Read the Dockerfile bytes.
	bytes, err := ioutil.ReadAll(dockerfile)
	if err != nil {
		return "", err
	}

	// Write the contents of the Dockerfile.
	_, err = fo.Write(bytes)
	if err != nil {
		return "", err
	}

	// Return the directory.
	return tempDir, nil
}

// extractBuildPackage extracts body into a temporary directory and returns the path.
func extractBuildPackage(body io.Reader, mimetype string) (string, error) {
	switch mimetype {
	case "application/zip", "application/x-zip-compressed":
		log.Info("buildpack identified as zip")
		dir, err := extractToTempDir(body, extractor.NewZip())
		if err != nil {
			return "", rpc.BuildPackError{Err: err.Error()}
		}
		return dir, nil

	case "application/x-tar", "application/gzip", "application/x-gzip":
		log.Info("buildpack identified as tar")
		dir, err := extractToTempDir(body, extractor.NewTgz())
		if err != nil {
			return "", rpc.BuildPackError{Err: err.Error()}
		}
		return dir, nil

	case "text/plain", "application/octet-stream":
		log.Info("buildpack identified as plain")
		dir, err := dockerfileTempDir(body)
		if err != nil {
			return "", rpc.BuildPackError{Err: err.Error()}
		}
		return dir, nil
	}

	return "", rpc.InvalidDockerfileError{Err: "Unsupported kind of build package: " + mimetype}
}

// Clone creates a temporary directory and `git clone`s a repository into it.
func Clone(url, sha, privateKey string) (string, error) {
	// Create a temp file for the ssh key.
	keyFile, err := ioutil.TempFile("", "ssh_key")
	if err != nil {
		return "", err
	}

	keyPath := keyFile.Name()

	// When this function is finished executing, close is guaranteed to execute
	// first, then the remove.
	defer os.Remove(keyPath)
	defer keyFile.Close()

	// Give the file the proper permissions.
	err = keyFile.Chmod(0600)
	if err != nil {
		return "", err
	}

	// Write the key to the file.
	_, err = io.WriteString(keyFile, privateKey)
	if err != nil {
		return "", err
	}

	// Create a temp directory to clone the buildpack into.
	bpPath, err := ioutil.TempDir("", "build_pack")
	if err != nil {
		return "", err
	}

	// In order to specify ssh keys per clone, we use option 1 of
	// https://gist.github.com/jzelinskie/1460b991a87220cc8adb
	// We assume that ssh-git.sh is located at the root of the filesystem.
	err = os.Setenv("GIT_SSH", "/ssh-git.sh")
	if err != nil {
		return "", err
	}
	err = os.Setenv("PKEY", keyPath)
	if err != nil {
		return "", err
	}

	// Clone into the temp directory by shelling out to git.
	output, err := timeoutActiveCommand("git", "clone", "--progress", url, bpPath)
	if err != nil {
		if err == ErrKilledInactiveProcess {
			return "", rpc.GitCloneError{Err: fmt.Sprintf("Timed out while trying to cloning git repository\n%s", output)}
		}
		return "", rpc.GitCloneError{Err: fmt.Sprintf("Error cloning git repository (%s)\n%s", err, output)}
	}
	log.Infof("git clone output: %s", output)

	// cd into the build package.
	// I really wish we didn't have to do this, but `git submodule` fails to find
	// the work tree when you give it envvars or parameters for GIT_DIR and
	// GIT_WORK_TREE.
	err = os.Chdir(bpPath)
	if err != nil {
		log.Fatalf("Error changing directory: %s", err)
	}

	// Defer cding into a directory that isn't a build package. Buildpack
	// directories are eventually deleted and we don't want to get any weird
	// behavior from executing in a removed directory.
	defer func() {
		err = os.Chdir("/")
		if err != nil {
			log.Fatalf("Error changing directory: %s", err)
		}
	}()

	// Checkout the specific SHA for the build.
	output, err = timeoutActiveCommand("git", "checkout", sha)
	if err != nil {
		if err == ErrKilledInactiveProcess {
			return "", rpc.GitCloneError{Err: fmt.Sprintf("Timed out while trying to checkout SHA %s in git repository\n%s", sha, output)}
		}
		return "", rpc.GitCheckoutError{Err: fmt.Sprintf("Error checking out git commit (%s)\n%s", err, output)}
	}
	log.Infof("git checkout output: %s", output)

	// Initialize any submodules. This will still have an exit code of 0 if there
	// are no submodules.
	output, err = timeoutCommand("git", "submodule", "update", "--init", "--recursive")
	if err != nil {
		if err == ErrKilledInactiveProcess {
			return "", rpc.GitCloneError{Err: fmt.Sprintf("Timed out while trying to update submodules in git repository\n%s", output)}
		}
		return "", rpc.GitCheckoutError{Err: fmt.Sprintf("Error initializing git submodules (%s): See submodule documentation at %s\n%s", err, quayDocsSubmoduleURL, output)}
	}
	log.Infof("git submodule output: %s", output)

	return bpPath, nil
}

type notifyingWriter struct {
	notifyChan chan error
	buf        *bytes.Buffer
}

func (w notifyingWriter) Write(p []byte) (n int, err error) {
	n, err = w.buf.Write(p)
	w.notifyChan <- err
	return
}

// ErrKilledInactiveProcess is used to indicate that a subprocess was killed
// due to a timeout.
var ErrKilledInactiveProcess = errors.New("killed process due to inactivity")

// timeoutCommand executes a command and kills the process if it doesn't exit
// before processTimeout. It should only be used if the command doesn't write
// frequently enough to standard out, thus timeoutActiveCommand cannot be used.
func timeoutCommand(command ...string) ([]byte, error) {
	if len(command) <= 0 {
		panic("buildpack: not enough arguments provided to timeoutCommand")
	}

	cmd := exec.Command(command[0], command[1:]...)

	type execResponse struct {
		data []byte
		err  error
	}

	done := make(chan *execResponse)
	go func() {
		data, err := cmd.CombinedOutput()
		done <- &execResponse{data, err}
	}()

	select {
	case resp := <-done:
		return resp.data, resp.err
	case <-time.Tick(processTimeout):
		log.Warningf("command `%v` timed out after %v\n", command, processTimeout)
		if err := cmd.Process.Kill(); err != nil {
			log.Fatalf("failed to kill long-running process: %s", err)
		}
		return nil, ErrKilledInactiveProcess
	}
}

// timeoutActiveCommand executes a commmand and kills the process if it doesn't
// output anything for more than the duration of processIdleTimeout.
// This function panics if you don't provide at least one string for commands.
func timeoutActiveCommand(command ...string) ([]byte, error) {
	if len(command) <= 0 {
		panic("buildpack: not enough arguments provided to timeoutActiveCommand")
	}

	cmd := exec.Command(command[0], command[1:]...)

	notifyChan := make(chan error, 1)
	notifyWriter := notifyingWriter{notifyChan, new(bytes.Buffer)}

	cmd.Stdout = &notifyWriter
	cmd.Stderr = &notifyWriter

	doneChan := make(chan error, 1)
	go func() {
		err := cmd.Run()
		doneChan <- err
	}()

	timeout := time.After(processIdleTimeout)
	for {
		select {
		case err := <-notifyChan:
			if err != nil {
				return notifyWriter.buf.Bytes(), err
			}

			timeout = time.After(processIdleTimeout)

		case <-timeout:
			log.Warningf("active command `%v` timed out after %v with output: %s\n", command, processTimeout, notifyWriter.buf.String())
			if err := cmd.Process.Kill(); err != nil {
				log.Fatalf("failed to kill hung process: %s", err)
			}
			return notifyWriter.buf.Bytes(), ErrKilledInactiveProcess

		case err := <-doneChan:
			return notifyWriter.buf.Bytes(), err
		}
	}
}
