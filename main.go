package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()

	node := NewNode(ctx)

	if node != nil {
		fmt.Println("Node initialized with peer ID:", node.ID().String())
	}

	go node.DiscoverNodes(ctx)

	select {}
}
