package client

import (
	"context"
	"fmt"

	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

func ExampleClient() {
	client, err := rmb.Default()
	if err != nil {
		panic(err)
	}

	node := NewNodeClient(10, client)

	_, err = node.Counters(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Println("ok")
	//Output: ok
}
