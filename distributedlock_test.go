package clusterleader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDistributedLockWithDefaultClient(t *testing.T) {

	// pre-set lock name
	lockname := "testlock"

	// set a consul token
	os.Setenv("CONSUL_HTTP_TOKEN", "acl-token")

	// create the first instance
	dlock, err := NewDistributedLockWithDefaultClient("locks", "localhost")
	assert.NoError(t, err)

	// create a second instance simulating another client
	dlock2, err := NewDistributedLockWithDefaultClient("locks", "secondhost")
	assert.NoError(t, err)

	// lock from first client
	err = dlock.Lock(lockname)
	assert.NoError(t, err)

	// second client should get an error because the lock is already held
	err = dlock2.Lock(lockname)
	assert.Error(t, err)

	// unlock
	err = dlock.Unlock(lockname)
	assert.NoError(t, err)

}
