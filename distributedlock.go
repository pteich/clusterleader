package clusterleader

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

type DistributedLock struct {
	mutex        sync.Mutex
	consulClient *api.Client
	prefix       string
	node         string
	locks        map[string]*api.Lock
}

// NewDistributedLock returns an instance with a given Consul client
func NewDistributedLock(consulClient *api.Client, prefix string, node string) (*DistributedLock, error) {
	return &DistributedLock{
		consulClient: consulClient,
		prefix:       strings.TrimSuffix(prefix, "/"),
		node:         node,
		locks:        make(map[string]*api.Lock),
	}, nil
}

// NewDistributedLockWithDefaultClient uses a Consul client with default settings (or values from ENV)
func NewDistributedLockWithDefaultClient(prefix string, node string) (*DistributedLock, error) {
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

	return NewDistributedLock(consulClient, prefix, node)
}

// Lock tries once to acquire the lock or return an error if it could not get it
func (dl *DistributedLock) Lock(key string) error {
	options := &api.LockOptions{
		Key:          dl.prefix + "/" + key,
		Value:        []byte(dl.node),
		LockTryOnce:  true,
		SessionTTL:   "10s",
		LockWaitTime: 15 * time.Millisecond,
	}

	lock, err := dl.consulClient.LockOpts(options)
	if err != nil {
		return errors.Wrap(err, "could not prepare lock")
	}

	closeChan, err := lock.Lock(nil)
	if err != nil {
		return errors.Wrap(err, "could not acquire lock")
	}

	if closeChan == nil {
		return errors.New("already locked")
	}

	dl.mutex.Lock()
	dl.locks[key] = lock
	dl.mutex.Unlock()

	return nil
}

// Unlock releases a specific lock
func (dl *DistributedLock) Unlock(key string) error {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()
	lock, found := dl.locks[key]
	if !found {
		return errors.New("lock not found")
	}

	// unlock and destroy the KV entry
	err := lock.Unlock()
	if err != nil {
		return errors.Wrap(err, "could not unlock")
	}
	err = lock.Destroy()
	if err != nil {
		return errors.Wrap(err, "could not remove lock")
	}

	delete(dl.locks, key)
	return nil
}

// UnlockAll releases all held locks
func (dl *DistributedLock) UnlockAll() {
	dl.mutex.Lock()
	for _, lock := range dl.locks {
		err := lock.Unlock()
		if err == nil {
			lock.Destroy()
		}
	}
	dl.locks = make(map[string]*api.Lock)
	dl.mutex.Unlock()
}