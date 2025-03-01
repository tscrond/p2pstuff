package p2pnode

import (
	"context"
	"log"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

type BootstrapNode struct {
	host.Host
}

func NewBootstrapNode(ctx context.Context) (*BootstrapNode, error) {
	keyFileName := "node_key.pem"

	nodeInfo, err := CreateNodeIdentity(keyFileName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	node, err := libp2p.New(
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/4001",
			"/ip6/::/tcp/4001",
		),
		libp2p.EnableRelay(),
		// libp2p.EnableRelayService(),
		// libp2p.NATPortMap(),
		libp2p.Identity(nodeInfo.Privkey),
		// libp2p.EnableHolePunching(), // Enables hole punching
	)
	if err != nil {
		log.Fatal("Failed to create host:", err)
		return nil, err
	}

	// Enable Circuit Relay v2 (Bootstrap acts as relay for NATed peers)
	_, err = relay.New(node)
	if err != nil {
		log.Fatal("Failed to start relay:", err)
	}

	kadDht, err := dht.New(ctx, node, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Println("Failed to create dht:", err)
		return nil, err
	}

	if err := kadDht.Bootstrap(ctx); err != nil {
		log.Println("Failed to bootstrap DHT:", err)
		return nil, err
	}
	
	return &BootstrapNode{Host: node}, nil
}
