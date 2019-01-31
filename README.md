# Cluster leader election using Consul distributed locks

```go
package main

import (
	"log"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pteich/clusterleader"
)

func main() {

	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"
	client, err := api.NewClient(config)

	clusterLeader, err := clusterleader.NewClusterleader(client, "testkey", "localhost", 15*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for err := range clusterLeader.Errors() {
			log.Print(err)
		}
	}()

	// as long as this loop runs we try to become leader
	// an isElected event is send on every state change
	// best would be to run this loop in a go routine and
	// react to whatever we are leader or not
	for isElected := range clusterLeader.Election() {
		if isElected {
			// we are leader
		} else {
			// not leader
		}
	}

}

```