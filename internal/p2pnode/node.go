package p2pnode

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

var CUSTOM_PEERS = []string{
	"/ip4/172.104.243.60/tcp/4001/p2p/12D3KooWJYueCcyg2CWyBmLeGFNUKYBvDskVHJdYS8L9UkbV9Dnm",
}

type NodeInfo struct {
	Privkey crypto.PrivKey `json:"priv_key"`
	Pubkey  crypto.PubKey  `json:"pub_key"`
}

type Node struct {
	host.Host
	BootstrapPeers   []peer.AddrInfo
	ActiveStreams    map[peer.ID]network.Stream
	streamLock       sync.Mutex
	protocolID       protocol.ID
	rendezvousString string
}

func NewNodeInfo(Privkey crypto.PrivKey) *NodeInfo {

	Pubkey := Privkey.GetPublic()
	return &NodeInfo{
		Privkey: Privkey,
		Pubkey:  Pubkey,
	}
}

func NewNode(ctx context.Context, rendezvousStr, protocolID, bootstrapMode string) (*Node, error) {
	keyFileName := "node_key.pem"

	protoID := protocol.ID(protocolID)

	nodeInfo, err := CreateNodeIdentity(keyFileName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	p2pHost, err := libp2p.New(
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays(getCustomPeers()),
		libp2p.EnableRelayService(),
		libp2p.Identity(nodeInfo.Privkey),
		libp2p.ForceReachabilityPrivate(),
		libp2p.DefaultSecurity,
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultPeerstore,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		log.Println("err creating p2p host:", err)
		return nil, err
	}

	var bootstrapPeers []peer.AddrInfo

	if bootstrapMode == "libp2p-default" {
		bootstrapPeers = getDefaultBootstrapPeers()
	} else if bootstrapMode == "custom" {
		bootstrapPeers = getCustomPeers()
	}

	fmt.Println(bootstrapPeers)

	node := &Node{Host: p2pHost,
		BootstrapPeers:   bootstrapPeers,
		ActiveStreams:    make(map[peer.ID]network.Stream),
		protocolID:       protoID,
		rendezvousString: rendezvousStr,
	}

	node.Host.SetStreamHandler(protoID, func(s network.Stream) {
		log.Println("📞 New stream from:", s.Conn().RemotePeer())

		buf := bufio.NewReader(s)
		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				log.Println("❌ Stream closed:", s.Conn().RemotePeer(), err)
				s.Close()
				return
			}

			splitted := strings.Split(line, ":")[0]
			if splitted == "KEEPALIVE" {
				log.Println("🔄 Received keepalive from", s.Conn().RemotePeer())
			}

			log.Println("📩 Received message:", line)
		}
	})

	go node.sendKeepalive(10 * time.Second)

	return node, nil
}

func getDefaultBootstrapPeers() []peer.AddrInfo {
	bootstrapPeers := make([]peer.AddrInfo, len(dht.DefaultBootstrapPeers))
	// fmt.Println(bootstrapPeers)
	for i, addr := range dht.DefaultBootstrapPeers {
		multiAddr, _ := multiaddr.NewMultiaddr(addr.String())
		// fmt.Println(multiAddr)
		peerInfo, _ := peer.AddrInfoFromP2pAddr(multiAddr)
		bootstrapPeers[i] = *peerInfo
	}

	return bootstrapPeers
}

func getCustomPeers() []peer.AddrInfo {
	var addrs []peer.AddrInfo
	for _, peeraddr := range CUSTOM_PEERS {
		bootstrapAddr, _ := multiaddr.NewMultiaddr(peeraddr)
		bootstrapPeer, _ := peer.AddrInfoFromP2pAddr(bootstrapAddr)
		addrs = append(addrs, *bootstrapPeer)
	}
	return addrs
}
func CreateNodeIdentity(keyFileName string) (*NodeInfo, error) {
	var nodeInfo *NodeInfo

	if _, err := os.Stat(keyFileName); errors.Is(err, os.ErrNotExist) {
		privkey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
		if err != nil {
			log.Printf("Failed to generate key pair: %v\n", err)
			return nil, err
		}

		nodeinfo := NewNodeInfo(privkey)

		if err := savePrivKey(*nodeinfo, keyFileName); err != nil {
			log.Println("err serializing node:", err)
			return nil, err
		}

		nodeInfo = nodeinfo

	} else {
		nodeinfo, err := readNodeInfoFromFile(keyFileName)
		if err != nil {
			log.Println("err reading node from file", err)
			return nil, err
		}

		nodeInfo = nodeinfo
	}

	return nodeInfo, nil
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

	// time.Sleep(5 * time.Second) // Give DHT time to stabilize

	log.Println("🌍 Node started with ID:", node.Host.ID())
	log.Println("📡 Listening on:", node.Host.Addrs())

	rendezvousString := node.rendezvousString

	routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, rendezvousString)
	// time.Sleep(3 * time.Second) // Wait for discovery to propagate

	go node.continuousFindPeers(ctx, *routingDiscovery, rendezvousString)

	return nil

}

func (node *Node) continuousFindPeers(ctx context.Context, routingDiscovery routing.RoutingDiscovery, rendezvousString string) {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// log.Println("🔄 Searching for new peers...")
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

		log.Printf("🌐 Connected peers: %d %s", len(node.Host.Network().Peers()), node.Host.Network().Peers())

		// log.Println("🔍 Found peer:", peerInfo.ID)
		// log.Println("🛜 Addresses:", peerInfo.Addrs)

		if len(peerInfo.Addrs) == 0 {
			// log.Println("⚠️ Peer has no known addresses, skipping")
			continue
		}

		if err := node.connectToPeer(ctx, peerInfo); err != nil {
			// log.Println("⚠️ Could not connect to peer:", err)
			continue
		}
	}

	return nil
}

func (node *Node) connectToPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	// fmt.Println("trying to connect to peer", peerInfo.ID)

	// fmt.Println("bbbebbebebebebeberb", node.Peerstore().PeerInfo(peerInfo.ID).ID.String())
	if node.Network().Connectedness(peerInfo.ID) == network.Connected {
		// log.Println("🔄 Already connected to", peerInfo.ID)
		return nil
	}

	// Context for connection AND stream
	connectCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Attempt to connect
	if err := node.Host.Connect(connectCtx, peerInfo); err != nil {
		return fmt.Errorf("could not connect: %w", err)
	}

	time.Sleep(500 * time.Millisecond)

	// fmt.Println("opening stream", peerInfo.ID)
	stream, err := node.Host.NewStream(connectCtx, peerInfo.ID, node.protocolID)
	if err != nil {
		return fmt.Errorf("stream open failed: %w", err)
	}
	log.Println("📡 Opened stream to peer:", peerInfo.ID)

	node.streamLock.Lock()
	node.ActiveStreams[peerInfo.ID] = stream
	node.streamLock.Unlock()

	log.Println("✅ Connected to peer:", peerInfo.ID)

	return nil
}

func (node *Node) sendKeepalive(interval time.Duration) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		// log.Println("🔄 Searching for new peers...")
		node.streamLock.Lock()
		log.Println("📡 Active streams:", len(node.ActiveStreams))
		for peerID, stream := range node.ActiveStreams {
			// log.Println("📡 Trying to send to:", peerID)
			if err := sendMessage(stream, fmt.Sprintf("KEEPALIVE:%s", peerID)); err != nil {
				log.Println("❌ Error sending keepalive to", peerID, ":", err)
				stream.Close()
				delete(node.ActiveStreams, peerID)
			} else {
				log.Println("Sent keepalive to", peerID)
			}
		}
		node.streamLock.Unlock()
	}

	// fmt.Println("sending hello msg", peerInfo.ID)

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
