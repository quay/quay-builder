package wamp

import (
	"sync"

	"gopkg.in/beatgammit/turnpike.v2"
)

// wampProtocolClient represents an implementation of interactions with the
// WAMP protocol.
type wampProtocolClient interface {
	JoinRealm(realm string, details map[string]interface{}) (map[string]interface{}, error)
	LeaveRealm() error
	Call(procedure string, args []interface{}, kwargs map[string]interface{}) (*turnpike.Result, error)
	Publish(topic string, args []interface{}, kwargs map[string]interface{}) error
	BasicRegister(procedure string, fn turnpike.BasicMethodHandler) error
	Close() error
}

// lockingWampClient defines a WAMP client that uses a lock to allow for
// multiple threads to safely send data via the internal client.
type lockingWampClient struct {
	tpClient *turnpike.Client
	sync.Mutex
}

// newLockingWampClient returns a WAMP client that uses a lock to allow
// multiples threads to *safely* send data via WAMP. This is necessary because
// the turnpike WAMP client currently interleaves the data of messages when
// sent via multiple goroutines simultaniously.
func newLockingWampClient(client *turnpike.Client) wampProtocolClient {
	return &lockingWampClient{
		tpClient: client,
	}
}

func (client *lockingWampClient) JoinRealm(realm string, details map[string]interface{}) (map[string]interface{}, error) {
	client.Lock()
	defer client.Unlock()
	return client.tpClient.JoinRealm(realm, details)
}

func (client *lockingWampClient) LeaveRealm() error {
	client.Lock()
	defer client.Unlock()
	return client.tpClient.LeaveRealm()
}

func (client *lockingWampClient) Call(procedure string, args []interface{}, kwargs map[string]interface{}) (*turnpike.Result, error) {
	client.Lock()
	defer client.Unlock()
	return client.tpClient.Call(procedure, args, kwargs)
}

func (client *lockingWampClient) Publish(topic string, args []interface{}, kwargs map[string]interface{}) error {
	client.Lock()
	defer client.Unlock()
	return client.tpClient.Publish(topic, args, kwargs)
}

func (client *lockingWampClient) BasicRegister(procedure string, fn turnpike.BasicMethodHandler) error {
	client.Lock()
	defer client.Unlock()
	return client.tpClient.BasicRegister(procedure, fn)
}

func (client *lockingWampClient) Close() error {
	client.Lock()
	defer client.Unlock()
	return client.tpClient.Close()
}
