package main

import (
	"bufio"
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
	"github.com/libp2p/go-libp2p/core/network"
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
	// fmt.Println(bootstrapPeers)
	for i, addr := range dht.DefaultBootstrapPeers {
		multiAddr, _ := multiaddr.NewMultiaddr(addr.String())
		// fmt.Println(multiAddr)
		peerInfo, _ := peer.AddrInfoFromP2pAddr(multiAddr)
		bootstrapPeers[i] = *peerInfo
	}

	node := &Node{Host: p2pHost, BootstrapPeers: bootstrapPeers}

	node.Host.SetStreamHandler("/bobaklabs/1.0.0", func(s network.Stream) {
		defer s.Close()
		fmt.Println("📞 New stream from:", s.Conn().RemotePeer())

		scanner := bufio.NewScanner(s)
		for scanner.Scan() {
			msg := scanner.Text()
			fmt.Println("📩 Received:", msg)
			_, _ = s.Write([]byte("ACK: " + msg + "\n")) // Send acknowledgment
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading from stream:", err)
		}
	})

	return node
}

func (node *Node) DiscoverNodes(ctx context.Context) error {

	// fmt.Println(node.BootstrapPeers)

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

	time.Sleep(5 * time.Second) // Give DHT time to stabilize

	log.Println("🌍 Node started with ID:", node.Host.ID())
	log.Println("📡 Listening on:", node.Host.Addrs())

	rendezvousString := "bobaklabs-rendezvous"

	routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, rendezvousString)
	time.Sleep(3 * time.Second) // Wait for discovery to propagate

	go node.continuousFindPeers(ctx, *routingDiscovery, rendezvousString)

	return nil

}

func (node *Node) continuousFindPeers(ctx context.Context, routingDiscovery routing.RoutingDiscovery, rendezvousString string) {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("🔄 Searching for new peers...")
			node.findPeers(ctx, routingDiscovery, rendezvousString) // Keep discovering peers
		case <-ctx.Done():
			log.Println("🛑 Stopping peer discovery")
			return
		}
	}
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

		log.Println("🔍 Found peer:", peerInfo.ID)
		// log.Println("🛜 Addresses:", peerInfo.Addrs)

		if len(peerInfo.Addrs) == 0 {
			// log.Println("⚠️ Peer has no known addresses, skipping")
			continue
		}

		if err := node.connectToPeer(ctx, peerInfo); err != nil {
			// log.Println("⚠️ Could not connect to peer:", err)
			continue
		}
		log.Println("✅ Connected to peer:", peerInfo.ID)
	}

	return nil
}

func (node *Node) connectToPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	fmt.Println("trying to connect to peer", peerInfo.ID)

	// Context for connection AND stream
	connectCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Attempt to connect
	if err := node.Host.Connect(connectCtx, peerInfo); err != nil {
		return fmt.Errorf("could not connect: %w", err)
	}

	fmt.Println("opening stream", peerInfo.ID)
	stream, err := node.Host.NewStream(connectCtx, peerInfo.ID, "/bobaklabs/1.0.0")
	if err != nil {
		return fmt.Errorf("stream open failed: %w", err)
	}
	defer stream.Close()

	fmt.Println("sending hello msg", peerInfo.ID)
	err = sendMessage(stream, fmt.Sprintf("NEW_PEER:%s", node.Host.ID()))
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}

	fmt.Println("reading responses", peerInfo.ID)
	scanner := bufio.NewScanner(stream)
	if scanner.Scan() {
		log.Println("📩 Received response:", scanner.Text())
	} else if err := scanner.Err(); err != nil {
		log.Println("Scanner error:", err)
	}

	return nil
}

func sendMessage(s network.Stream, message string) error {
	_, err := s.Write([]byte(message + "\n")) // Ensure messages are newline-delimited
	if err != nil {
		log.Println("Stream write error:", err)
		return err
	}
	log.Println("📨 Sent message:", message)
	return nil
}
