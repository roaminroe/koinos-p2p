package p2p

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"time"

	types "github.com/koinos/koinos-types-golang"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// GetInfo returns a test string
func GetInfo() string {
	return "test"
}

// GetNumber returns a test number
func GetNumber() types.UInt64 {
	return types.UInt64(10)
}

// KoinosP2PHost is the core object representing
type KoinosP2PHost struct {
	Host      host.Host
	Inventory NodeInventory
}

// NewKoinosP2PHost creates a libp2p host object listening on the given multiaddress
// uses secio encryption on the wire
// listenAddr is a multiaddress string on which to listen
// seed is the random seed to use for key generation. Use a negative number for a random seed.
func NewKoinosP2PHost(listenAddr string, seed int64) (*KoinosP2PHost, error) {
	var r io.Reader
	if seed == 0 {
		r = crand.Reader
	} else {
		r = mrand.New(mrand.NewSource(seed))
	}

	privateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	options := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.Identity(privateKey),
	}

	host, err := libp2p.New(context.Background(), options...)
	if err != nil {
		return nil, err
	}

	kHost := KoinosP2PHost{Host: host}

	return &kHost, nil
}

// ConnectToPeer connects the node to the given peer
func (n KoinosP2PHost) ConnectToPeer(peerAddr string) (*peerstore.AddrInfo, error) {
	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return nil, err
	}
	peer, err := peerstore.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := n.Host.Connect(ctx, *peer); err != nil {
		return nil, err
	}

	return peer, nil
}

// MakeContext creates and returns the canonical context which should be used for peer connections
// TODO: create this from configuration
func (n KoinosP2PHost) MakeContext() (ctx context.Context, cancel context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// GetListenAddress returns the multiaddress on which the node is listening
func (n KoinosP2PHost) GetListenAddress() multiaddr.Multiaddr {
	return n.Host.Addrs()[0]
}

// GetPeerAddress returns the ipfs multiaddress to which other peers should connect
func (n KoinosP2PHost) GetPeerAddress() multiaddr.Multiaddr {
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", n.Host.ID().Pretty()))
	return n.GetListenAddress().Encapsulate(hostAddr)
}

// Close closes the node
func (n KoinosP2PHost) Close() error {
	if err := n.Host.Close(); err != nil {
		return err
	}

	return nil
}