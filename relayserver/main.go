package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"

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
	if len(ha.Addrs()) > 0 {
		addr := ha.Addrs()[0]
		return addr.Encapsulate(hostAddr).String()
	} else {
		return hostAddr.String()
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle flags
	seedF := flag.Int64("seed", 0, "set random seed for id generation")
	listenF := flag.Int("l", 8919, "wait for incoming connections")
	flag.Parse()

	// setup the relay host - with  nonzero seed will haave a fixed address
	var r io.Reader
	if *seedF == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(*seedF))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		log.Printf("Failed to generate key pair: %v", err)
		return
	}
	// Create a host to act as a middleman to relay messages on our behalf
	relayHost, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *listenF)),
		libp2p.Identity(priv),
	)
	if err != nil {
		log.Printf("Failed to create relayHost: %v", err)
		return
	}

	fullAddr := getHostAddress(relayHost)
	log.Printf("Relay is: %s\nUse this address in setting up relay services", fullAddr)

	// Configure the host to offer the ciruit relay service.
	// Any host that is directly dialable in the network (or on the internet)
	// can offer a circuit relay service, this isn't just the job of
	// "dedicated" relay services.
	// In circuit relay v2 (which we're using here!) it is rate limited so that
	// any node can offer this service safely
	// do so with an option that keeps the stream alive indefinitely
	_, err = relay.New(relayHost, relay.WithInfiniteLimits())
	if err != nil {
		log.Printf("Failed to instantiate the relay: %v", err)
		return
	}

	/*
		// we want to keep looking at its peerstrore
		go func() {
			// old := peer.IDSlice{}
			for {
				time.Sleep(1 * time.Minute)
				idslice := relayHost.Peerstore().PeersWithAddrs()
				// if len(idslice) == len(old) {
				// 	continue
				// }
				log.Printf("ids: %v\n", idslice)
				// old = idslice
			}
		}()

	*/

	// Run until canceled.
	<-ctx.Done()
}
