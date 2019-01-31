package clusterleader

import (
	"net"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

// Clusterleader uses Consul distributed locks to elect a cluster leader
type Clusterleader struct {
	sync.Mutex
	consulClient *api.Client
	key          string
	node         string
	leader       bool
	lock         *api.Lock
	lockChan     <-chan struct{}
	stopChan     chan struct{}
	electionChan chan bool
	errorChan    chan error
	waitTime     time.Duration
}

// NewClusterleader returns a new Clusterleader instance with a given consul connection
func NewClusterleader(consulClient *api.Client, key string, node string, waitTime time.Duration) (*Clusterleader, error) {

	return &Clusterleader{
		consulClient: consulClient,
		key:          key,
		node:         node,
		leader:       false,
		stopChan:     make(chan struct{}),
		electionChan: make(chan bool),
		errorChan:    make(chan error),
		waitTime:     waitTime,
	}, nil

}

// NewClusterLeaderWithDefaultClient uses a default Consul client (which can also be set using Consul env variables)
func NewClusterleaderWithDefaultClient(key string, node string, waitTime time.Duration) (*Clusterleader, error) {

	dialContext := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 15 * time.Second,
	}).DialContext

	consulCfg := api.DefaultConfig()
	consulCfg.Transport.DialContext = dialContext

	consulClient, err := api.NewClient(consulCfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize Consul client")
	}

	return NewClusterleader(consulClient, key, node, waitTime)
}

// Election returns a channel that signals if we are leader or not
func (cl *Clusterleader) Election() <-chan bool {

	go func() {

		defer close(cl.electionChan)
		defer close(cl.errorChan)

		for {

			// first set non leader status
			// if we can not get the lock, getLock() blocks until
			// we get it or returns an error and tries again after
			// a given wait time
			cl.updateStatus(false)
			err := cl.getLock()
			if err != nil {
				cl.errorChan <- err
				time.Sleep(cl.waitTime)
				continue
			}

			// we got the lock and are leader now
			cl.updateStatus(true)

			// wait for either the lock goes away or we get
			// an explicit stop
			select {
			case <-cl.lockChan:
				cl.lock.Unlock()

			case <-cl.stopChan:
				cl.lock.Unlock()
				return
			}
		}

	}()

	return cl.electionChan
}

// Errors returns a channel that receives all errors
func (cl *Clusterleader) Errors() <-chan error {
	return cl.errorChan
}

// IsLeader just returns if we are leader or not
func (cl *Clusterleader) IsLeader() bool {
	return cl.leader
}

// Stop ends the leader election
func (cl *Clusterleader) Stop() {
	close(cl.stopChan)
}

func (cl *Clusterleader) getLock() error {

	options := &api.LockOptions{
		Key:        cl.key,
		Value:      []byte(cl.node),
		SessionTTL: cl.waitTime.String(),
	}

	lock, err := cl.consulClient.LockOpts(options)
	if err != nil {
		return errors.Wrap(err, "could not prepare lock")
	}

	// acquire the lock and return a channel that is closed upon lost
	lockChan, err := lock.Lock(cl.stopChan)
	if err != nil {
		return errors.Wrap(err, "could not acquire lock")
	}

	cl.lockChan = lockChan
	cl.lock = lock

	return nil
}

func (cl *Clusterleader) updateStatus(isLeader bool) {
	cl.Lock()
	defer cl.Unlock()
	cl.leader = isLeader
	cl.electionChan <- isLeader
}
