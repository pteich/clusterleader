package clusterleader

import (
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
)

func TestClusterleader_Election(t *testing.T) {
	testConsul, err := testutil.NewTestServerConfigT(t, func(c *testutil.TestServerConfig) {
		c.Bootstrap = true
		c.ACLDefaultPolicy = "allow"
	})
	assert.NoError(t, err)
	defer testConsul.Stop()

	config := api.DefaultConfig()
	config.Address = testConsul.HTTPAddr
	client, err := api.NewClient(config)
	assert.NoError(t, err)

	clusterLeader, err := NewClusterleader(client, "testkey", "localhost", 15*time.Second)
	assert.NoError(t, err)

	go func() {
		for err := range clusterLeader.Errors() {
			t.Log(err)
		}
	}()

	time.AfterFunc(5*time.Second, func() {
		clusterLeader.Stop()
	})

	for isElected := range clusterLeader.Election() {
		if isElected {
			t.Log("leader")
		} else {
			t.Log("not leader")
		}
	}
}
