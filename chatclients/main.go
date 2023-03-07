package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"

	golog "github.com/ipfs/go-log/v2"
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

	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(golog.LevelInfo) // Change to INFO for extra info

	// flags
	relayF := flag.String("r", "", "relay host full address")
	mirrorF := flag.String("m", "", "mirror receiver host full address")
	flag.Parse()

	if *relayF == "" {
		log.Fatalf("use -r to set relay host full address")
	}

	// define a host that is unreachable
	hs, err := libp2p.New(
		libp2p.NoListenAddrs,
		// Usually EnableRelay() is not required as it is enabled by default
		// but NoListenAddrs overrides this, so we're adding it in explictly again.
		libp2p.EnableRelay(),
	)
	if err != nil {
		log.Printf("failed to produce host: %v\n", err)
		return
	}
	fullLAddr := getHostAddress(hs)
	log.Printf("I am host: %s\n", fullLAddr)

	// Extract the relay peer ID from the multiaddr.
	relayinfo, err := full2info(*relayF)
	if err != nil {
		log.Println(err)
		return
	}

	if err := hs.Connect(context.Background(), *relayinfo); err != nil {
		log.Printf("Failed to connect receiver and relayhost: %v", err)
		return
	}

	if *mirrorF == "" { // we are a receiver
		doReceiver(hs, relayinfo)
	} else {
		doSender(hs, relayinfo, *mirrorF)
	}
	// wait for connections, close with ^C
	<-ctx.Done()
}

// setup a receiver host ( no -m flag)
func doReceiver(receiver host.Host, relayinfo *peer.AddrInfo) {
	// set up a protocol handler on receiver
	receiver.SetStreamHandler("/chat/1.0.0", func(s network.Stream) {
		log.Println("Awesome! We're now communicating via the relay!")

		// Create a buffer stream for non blocking read and write.
		rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

		go readData(rw)
		go writeData(rw)
	})

	// Hosts that want to have messages relayed on their behalf need to reserve a slot
	// with the circuit relay service host
	// As we will open a stream to receiver, receiver needs to make the
	// reservation
	_, err := client.Reserve(context.Background(), receiver, *relayinfo)
	if err != nil {
		log.Printf("receiver failed to obtain a relay reservation from relayhost. %v", err)
		return
	}
}

// setup a sender host
func doSender(sender host.Host, relayinfo *peer.AddrInfo, fullLAddr string) {
	// Now create a new address for receiver that specifies to communicate via
	// relayhost using a circuit relay
	relayaddr, err := ma.NewMultiaddr("/p2p/" + relayinfo.ID.String() + "/p2p-circuit" + fullLAddr)
	if err != nil {
		log.Println(err)
		return
	}

	// we want sender to have access to listerner ID from full address:
	receiverinfo, err := full2info(fullLAddr)
	if err != nil {
		log.Println(err)
		return
	}

	idArr := peer.AddrInfosToIDs([]peer.AddrInfo{*receiverinfo})
	receiverID := idArr[0]

	// modify receiver's connection using the relay
	receiverrelayinfo := peer.AddrInfo{
		ID:    receiverID,                //receiver.ID(),
		Addrs: []ma.Multiaddr{relayaddr}, // ^^^ assumes we have the receiver object
	}

	// here the sender connects to listerner
	if err := sender.Connect(context.Background(), receiverrelayinfo); err != nil {
		log.Printf("Unexpected error here. Failed to connect sender and receiver: %v", err)
		return
	}

	// Because we don't have a direct connection to the destination node - we have a relayed connection -
	// the connection is marked as transient. Since the relay limits the amount of data that can be
	// exchanged over the relayed connection, the application needs to explicitly opt-in into using a
	// relayed connection. In general, we should only do this if we have low bandwidth requirements,
	// and we're happy for the connection to be killed when the relayed connection is replaced with a
	// direct (holepunched) connection.
	s, err := sender.NewStream(network.WithUseTransient(context.Background(), "/chat/1.0.0"), receiverID, "/chat/1.0.0")
	if err != nil {
		log.Println("Whoops, this should have worked...: ", err)
		return
	}

	// Create a buffered stream so that read and writes are non blocking.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	// Create a thread to read and write data.
	go writeData(rw)
	go readData(rw)
}

// convert string multiaddres into peer.Addrinfo
func full2info(fullAddr string) (*peer.AddrInfo, error) {
	// Turn the fulladdr into a multiaddr.
	maddr, err := ma.NewMultiaddr(fullAddr)
	if err != nil {
		// log.Println(err)
		return nil, err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		rw.WriteString(fmt.Sprintf("%s\n", sendData))
		rw.Flush()
	}
}
