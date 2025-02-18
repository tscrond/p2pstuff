package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

type NodeInfo struct {
	Privkey crypto.PrivKey `json:"priv_key"`
	Pubkey  crypto.PubKey  `json:"pub_key"`
}

type Node struct {
	BootstrapPeers []peer.AddrInfo
	host.Host
}

func NewNodeInfo(
	Privkey crypto.PrivKey) *NodeInfo {

	Pubkey := Privkey.GetPublic()
	return &NodeInfo{
		Privkey: Privkey,
		Pubkey:  Pubkey,
	}
}

func NewNode(ctx context.Context) *Node {
	var nodeInfo NodeInfo
	keyFileName := "node_key.pem"

	if _, err := os.Stat(keyFileName); errors.Is(err, os.ErrNotExist) {
		privkey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
		if err != nil {
			log.Printf("Failed to generate key pair: %v\n", err)
			return nil
		}

		nodeinfo := NewNodeInfo(privkey)

		if err := savePrivKey(*nodeinfo, keyFileName); err != nil {
			log.Println("err serializing node:", err)
			return nil
		}

		nodeInfo = *nodeinfo

	} else {
		nodeinfo, err := readNodeInfoFromFile(keyFileName)
		if err != nil {
			log.Println("err reading node from file", err)
			return nil
		}

		nodeInfo = *nodeinfo
	}

	p2pHost, err := libp2p.New(
		libp2p.Identity(nodeInfo.Privkey),
		libp2p.DefaultSecurity,
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultPeerstore,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		log.Println("err creating p2p host:", err)
		return nil
	}

	bootstrapPeers := make([]peer.AddrInfo, len(dht.DefaultBootstrapPeers))
	fmt.Println(bootstrapPeers)
	for i, addr := range dht.DefaultBootstrapPeers {
		multiAddr, _ := multiaddr.NewMultiaddr(addr.String())
		fmt.Println(multiAddr)
		peerInfo, _ := peer.AddrInfoFromP2pAddr(multiAddr)
		bootstrapPeers[i] = *peerInfo
	}

	node := &Node{Host: p2pHost, BootstrapPeers: bootstrapPeers}

	return node
}

func (node *Node) DiscoverNodes(ctx context.Context) error {

	fmt.Println(node.BootstrapPeers)

	kademliaDHT, err := dht.New(ctx, node.Host, dht.BootstrapPeers(node.BootstrapPeers...))
	if err != nil {
		log.Println(err)
		return nil
	}

	fmt.Println("🌍 Bootstrapping the DHT")
	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		log.Println(err)
		return nil
	}

	time.Sleep(1 * time.Second)

	log.Println("🌍 Node started with ID:", node.Host.ID())
	log.Println("📡 Listening on:", node.Host.Addrs())

	rendezvousString := "bobaklabs-rendezvous"

	routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, rendezvousString)

	return node.findPeers(ctx, *routingDiscovery, rendezvousString)
}

func (node *Node) findPeers(ctx context.Context, routingDiscovery routing.RoutingDiscovery, rendezvousString string) error {
	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvousString)
	if err != nil {
		log.Println(err)
		return err
	}

	for peerInfo := range peerChan {
		if peerInfo.ID == node.Host.ID() {
			continue
		}

		log.Println("🔍 Found peer: ", peerInfo.ID)

		// if err := node.Host.Connect(ctx, peerInfo); err != nil {
		// 	log.Println("⚠️ Could not connect to peer:", err)
		// 	continue
		// }

		if err := node.connectToPeer(ctx, peerInfo); err != nil {
			log.Println("⚠️ Could not connect to peer:", err)
			continue
		}

		log.Println("✅ Connected to peer:", peerInfo.ID)
	}

	return nil
}

func (node *Node) connectToPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	if err := node.Host.Connect(ctx, peerInfo); err != nil {
		return fmt.Errorf("could not connect: %w", err)
	}

	stream, err := node.Host.NewStream(ctx, peerInfo.ID, "/blockPoC/1.0.0")
	if err != nil {
		return fmt.Errorf("stream write failed: %w", err)
	}

	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		return fmt.Errorf("stream read failed: %w", err)
	}
	log.Println("Received:", string(buf[:n]))

	return nil
}
