package clusterleader

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDistributedLockWithDefaultClient(t *testing.T) {

	// pre-set lock name
	lockname := "testlock"

	// set a consul token
	os.Setenv("CONSUL_HTTP_TOKEN", "acl-token")

	// create the first instance
	dlock, err := NewDistributedLockWithDefaultClient("locks", 30*time.Second)
	assert.NoError(t, err)

	// create a second instance simulating another client
	dlock2, err := NewDistributedLockWithDefaultClient("locks", 30*time.Second)
	assert.NoError(t, err)

	// lock from first client
	_, err = dlock.Lock(lockname, "host1", 1*time.Second)
	assert.NoError(t, err)

	// second client should get an error because the lock is already held
	_, err = dlock2.Lock(lockname, "host2", 1*time.Second)
	assert.Error(t, err)

	// unlock
	err = dlock.Unlock(lockname)
	assert.NoError(t, err)

}
