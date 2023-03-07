package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"

	ma "github.com/multiformats/go-multiaddr"
)

func getHostAddress(ha host.Host) string {
	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", ha.ID().String()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := ha.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// var r io.Reader
	// if randseed == 0 {
	var r = rand.Reader
	// } else {
	// 	r = mrand.New(mrand.NewSource(randseed))
	// }

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		log.Printf("Failed to generate key pair: %v", err)
		return
	}

	// Create a host to act as a middleman to relay messages on our behalf
	relay1, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 8919)),
		libp2p.Identity(priv),
	)
	if err != nil {
		log.Printf("Failed to create relay1: %v", err)
		return
	}

	fullAddr := getHostAddress(relay1)

	// Configure the host to offer the ciruit relay service.
	// Any host that is directly dialable in the network (or on the internet)
	// can offer a circuit relay service, this isn't just the job of
	// "dedicated" relay services.
	// In circuit relay v2 (which we're using here!) it is rate limited so that
	// any node can offer this service safely
	_, err = relay.New(relay1)
	if err != nil {
		log.Printf("Failed to instantiate the relay: %v", err)
		return
	}

	log.Printf("Relay is: %s\nUse this address in setting up relay services", fullAddr)

	// Run until canceled.
	<-ctx.Done()
}
