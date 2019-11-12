package wamp

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	turnpike "gopkg.in/beatgammit/turnpike.v2"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"

	"github.com/quay/quay-builder/rpc"
)

const (
	// connectMaxRetries represents the maximum number of retries for trying to
	// connect to a build manager.
	connectMaxRetries = 5

	// heartbeatInterval is duration waited before a worker sends a heartbeat
	// RPC.
	heartbeatInterval = time.Second

	// connectTimeout is the number of seconds to wait before considering a
	// websocket connection failed.
	connectTimeout = 10 * time.Second
)

type wampError struct {
	URI turnpike.URI
	Err error
}

func (we wampError) Error() string {
	return we.Err.Error()
}

// errFailedToAllocateRealm is returned when a BuildManager fails to return a new
// realm.
var errFailedToAllocateRealm = errors.New("failed to get realm from BuildManager")

type wampClient struct {
	realm        string
	token        string
	currentPhase rpc.Phase
	tlsConfig    *tls.Config
	protocol     wampProtocolClient
}

// NewClient returns a new implementation of rpc.Client using the WAMP protocol
// as the underlying transport.
func NewClient(realm, token string, tlscfg *tls.Config) (rpc.Client, error) {
	return &wampClient{
		realm:     realm,
		token:     token,
		tlsConfig: tlscfg,
	}, nil
}

func (c *wampClient) SetPhase(phase rpc.Phase, pmd *rpc.PullMetadata) error {
	c.currentPhase = phase
	data := make(map[string]interface{})
	data["status_data"] = make(map[string]interface{})
	if pmd != nil {
		data["status_data"] = map[string]interface{}{
			"registry":       pmd.RegistryURL,
			"base_image":     pmd.BaseImage,
			"base_image_tag": pmd.BaseImageTag,
			"pull_username":  pmd.PullUsername,
		}
	}

	jsonData, err := json.Marshal(&data)
	if err != nil {
		log.Panicf("Error when marshaling status data: %v", err)
	}

	kwargs := make(map[string]interface{})
	kwargs["phase"] = phase
	kwargs["json_data"] = string(jsonData)
	result, err := c.protocol.Call("io.quay.builder.logmessagesynchronously", make([]interface{}, 0), kwargs)
	if err != nil {
		return err
	}
	if len(result.Arguments) == 0 {
		return wampError{
			"io.quay.builder.errorduringphasetransition",
			errors.New("failed to get response from phase transition"),
		}
	}

	approved, ok := result.Arguments[0].(bool)
	if !ok {
		return wampError{
			"io.quay.builder.errorduringphasetransition",
			errors.New("failed to cast turnpike result to bool"),
		}
	}

	if !approved {
		return rpc.ErrClientRejectedPhaseTransition{Err: "Phase transition was rejected"}
	}

	return nil
}

func (c *wampClient) RegisterBuildCallback(callback rpc.BuildCallback) error {
	return c.protocol.BasicRegister(
		"io.quay.builder.build",
		func(args []interface{}, kwargs map[string]interface{}) *turnpike.CallResult {
			// Parse the BuildArgs.
			var buildArgs rpc.BuildArgs
			if err := mapstructure.Decode(kwargs, &buildArgs); err != nil {
				log.Errorf("error decoding build args: %s", err)
				return &turnpike.CallResult{
					Args:   make([]interface{}, 0, 0),
					Kwargs: make(map[string]interface{}),
					Err:    "io.quay.builder.missingorinvalidargument",
				}
			}
			log.Infof("decoded build args: %#v", buildArgs)

			// Call the callback.
			buildMetadata, err := callback(c, &buildArgs)
			kwargs = make(map[string]interface{})
			var uri turnpike.URI
			if err != nil {
				// Report the error to the BuildManager.
				log.Infof("build resulted in an error: %s", err.Error())
				kwargs["base_error"] = err.Error()
				uri = uriFromError(err)

			} else {
				log.Infof("build completed successfully: %#v", buildMetadata)
				kwargs["image_id"] = buildMetadata.ImageID
				kwargs["digests"] = buildMetadata.Digests
			}

			return &turnpike.CallResult{
				Args:   make([]interface{}, 0),
				Kwargs: kwargs,
				Err:    uri,
			}
		},
	)
}

func uriFromError(err error) turnpike.URI {
	var uri turnpike.URI = "io.quay.builder.internalerror"
	switch typedErr := err.(type) {
	case rpc.GitCheckoutError:
		uri = "io.quay.builder.gitcheckout"
	case rpc.GitCloneError:
		uri = "io.quay.builder.gitfailure"
	case rpc.BuildPackError:
		uri = "io.quay.builder.buildpackissue"
	case rpc.CannotPullForCacheError:
		uri = "io.quay.builder.cannotpullforcache"
	case rpc.TagError:
		uri = "io.quay.builder.tagissue"
	case rpc.PushError:
		uri = "io.quay.builder.pushissue"
	case rpc.PullError:
		uri = "io.quay.builder.cannotpullbaseimage"
	case rpc.BuildError:
		uri = "io.quay.builder.builderror"
	case rpc.InvalidDockerfileError:
		uri = "io.quay.builder.dockerfileissue"
	case rpc.ErrClientRejectedPhaseTransition:
		uri = "io.quay.builder.clientrejectedtransition"
	case wampError:
		uri = typedErr.URI
	default:
		log.Warningf("Could not find matching error URI for error (type: %T) '%v'", err, err)
	}

	log.Infof("determined final WAMP response URI for error (type: %T) '%v': %s", err, err, uri)
	return uri
}

func (c *wampClient) FindMostSimilarTag(md rpc.TagMetadata) (string, error) {
	kwargs := make(map[string]interface{})
	kwargs["base_image_name"] = md.BaseImage
	kwargs["base_image_tag"] = md.BaseImageTag
	kwargs["base_image_id"] = md.BaseImageID
	kwargs["command_comments"] = "" // This field is no longer used.
	result, err := c.protocol.Call("io.quay.buildworker.determinecachetag", make([]interface{}, 0, 0), kwargs)
	if err != nil {
		return "", err
	}

	if len(result.Arguments) == 0 {
		return "", rpc.ErrNoSimilarTags
	}

	tag, ok := result.Arguments[0].(string)
	if !ok {
		return "", wampError{
			"io.quay.builder.cachelookupissue",
			errors.New("failed to cast turnpike result to string"),
		}
	}

	return tag, nil
}

func (c *wampClient) PublishBuildLogEntry(entry string) error {
	kwargs := make(map[string]interface{})
	kwargs["json_data"] = entry
	kwargs["phase"] = c.currentPhase
	_, err := c.protocol.Call("io.quay.builder.logmessagesynchronously", make([]interface{}, 0), kwargs)
	return err
}

func (c *wampClient) ListenAndServe() {
	// Notify the BuildManager that the worker is ready to receive the build RPC.
	go func() {
		args := make(map[string]interface{})
		args["token"] = c.token
		args["version"] = "0.3"

		// This RPC doesn't return a response until after the manager calls the
		// registered build callback.
		_, err := c.protocol.Call("io.quay.buildworker.ready", make([]interface{}, 0, 0), args)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Block indefinitely sending heartbeats every heartbeatInterval.
	for {
		if c.protocol != nil {
			log.Info("attempting heartbeat to a BuildManager")
			err := c.protocol.Publish("io.quay.builder.heartbeat", make([]interface{}, 0, 0), make(map[string]interface{}))
			if err != nil {
				log.Fatalf("failed to heartbeat to a BuildManager: %v", err)
			}
			log.Info("successfully sent heartbeat to BuildManager")
		}

		time.Sleep(heartbeatInterval)
	}
}

func (c *wampClient) Connect(endpoint string) error {
	// Attempt to connect to the BuildManager with expotential backoff.
	var attempts float64
	for {
		err := c.connect(endpoint)
		if err == nil {
			break
		}

		if attempts >= connectMaxRetries-1 {
			log.Fatal(err)
		}

		attempts++
		seconds := int64(math.Pow(2, attempts))
		log.Warningf("Got error when connecting to build manager: %v", err)
		log.Infof("Sleeping for %v seconds (attempt #%v of %v)", seconds, attempts, connectMaxRetries)
		time.Sleep(time.Duration(seconds) * time.Second)
	}

	return nil
}

type potentialClient struct {
	tpClient *turnpike.Client
	err      error
}

// timeoutTurnpikeConn attempts to establish a turnpike.Client connection and
// fails if the connection is not successful within the provided timeout.
func timeoutTurnpikeConn(endpoint string, tlsConfig *tls.Config, timeout time.Duration) (*turnpike.Client, error) {
	wsChan := make(chan *potentialClient)
	go func() {
		log.Infof("attempting to connect to websocket %s", endpoint)
		tpClient, err := turnpike.NewWebsocketClient(turnpike.JSON, endpoint, tlsConfig)
		if err != nil {
			wsChan <- &potentialClient{nil, err}
		}
		wsChan <- &potentialClient{tpClient, nil}
	}()

	select {
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out after attempting to establish websocket connection for %s", timeout)
	case pws := <-wsChan:
		return pws.tpClient, pws.err
	}
}

func (c *wampClient) connect(endpoint string) error {
	// Attempt to connect to a Build Manager, with a "black box timeout".
	tpClient, err := timeoutTurnpikeConn(endpoint, c.tlsConfig, connectTimeout)
	if err != nil {
		return err
	}

	// Make the turnpike.Client thread-safe.
	c.protocol = newLockingWampClient(tpClient)

	// If there wasn't a pre-allocated realm, get one from a BuildManager.
	if c.realm == "" {
		log.Info("dynamically registering worker")

		// Join the registration realm.
		err := joinRealm(c.protocol, "registration")
		if err != nil {
			return err
		}

		// Call the register RPC.
		result, err := c.protocol.Call("io.quay.buildworker.register", make([]interface{}, 0, 0), make(map[string]interface{}))
		if err != nil {
			return err
		}
		if len(result.Arguments) == 0 {
			// Fail if we get nothing as a response.
			return errFailedToAllocateRealm
		}

		// Attempt to cast the response to a string.
		var ok bool
		c.realm, ok = result.Arguments[0].(string)
		if !ok {
			return errFailedToAllocateRealm
		}

		// Leave the registration realm.
		err = c.protocol.LeaveRealm()
		if err != nil {
			return err
		}
	}

	// Join the realm.
	err = joinRealm(c.protocol, c.realm)
	if err != nil {
		return err
	}

	// Ping to ensure the realm is there and working.
	_, err = c.protocol.Call("io.quay.buildworker.ping", make([]interface{}, 0, 0), make(map[string]interface{}))
	return err
}

func joinRealm(protocol wampProtocolClient, realm string) error {
	log.Infof("joining realm %s", realm)
	_, err := protocol.JoinRealm(realm, make(map[string]interface{}))
	return err
}
