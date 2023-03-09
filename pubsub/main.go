package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

var (
	topicNameF = flag.String("topicName", "chinangwa", "name of topic to join")
	portF      = flag.Int("p", 0, "port to use")
)

type application struct {
	store, restore map[string]string
	found          chan peer.AddrInfo
}

func main() {
	flag.Parse()
	ctx := context.Background()

	listener := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *portF)
	h, err := libp2p.New(libp2p.ListenAddrStrings(listener))
	if err != nil {
		panic(err)
	}

	myStore := make(map[string]string)
	myReStore := make(map[string]string)
	foundPeer := make(chan peer.AddrInfo)

	app := application{
		store:   myStore,
		restore: myReStore,
		found:   foundPeer,
	}

	go app.discoverPeers(ctx, h)

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	topic, err := ps.Join(*topicNameF)
	if err != nil {
		panic(err)
	}

	// groutine that sends out messages
	go streamConsoleTo(ctx, topic)

	sub, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}

	// set the app.store
	go func(app *application) {
		for {
			p := <-app.found
			// count from 1
			i := 0
			for range app.store {
				i++
			}
			label := fmt.Sprintf("%d", i+1)
			app.store[p.ID.String()] = label
			app.restore[label] = p.ID.String() // do we want a map into peers?
		}
	}(&app)

	// // we want to keep looking at attached hosts
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			println("--------------------------")
			log.Println("mystore:")
			for p := range myStore {
				println(p) // log.Printf("id: %s\n", p)
			}
			println("--------------------------")
		}
	}()

	// loop that prints responses
	app.printMessagesFrom(ctx, sub)
}

func initDHT(ctx context.Context, h host.Host) *dht.IpfsDHT {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		panic(err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				fmt.Println("Bootstrap warning:", err)
			}
		}()
	}
	wg.Wait()

	return kademliaDHT
}

func (app *application) discoverPeers(ctx context.Context, h host.Host) {
	kademliaDHT := initDHT(ctx, h)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, *topicNameF)

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	for !anyConnected {
		fmt.Println("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, *topicNameF)
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				fmt.Println("Failed connecting to ", peer.ID.String(), ", error:", err)
			} else {
				fmt.Println("Connected to:", peer.ID.String())
				// app.store[peer.ID.String()] = true
				app.found <- peer
				anyConnected = true
			}
		}
	}
	fmt.Println("Peer discovery complete")

	go app.fetchMore(ctx, routingDiscovery, h)
}

// lonng term search for peers
func (app *application) fetchMore(ctx context.Context, routingDiscovery *drouting.RoutingDiscovery, h host.Host) {
	for {
		//fmt.Println("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, *topicNameF)
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			peerAD := peer.ID.String()
			// test connectivity
			err := h.Connect(ctx, peer)
			if err != nil {
				if app.store[peerAD] != "" {
					delete(app.store, peerAD) // clear departed
				}
			} else {
				// silently add new found
				// app.store[peerAD] = true
				app.found <- peer
			}
		}
	}
}

func streamConsoleTo(ctx context.Context, topic *pubsub.Topic) {
	reader := bufio.NewReader(os.Stdin)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if err := topic.Publish(ctx, []byte(s)); err != nil {
			fmt.Println("### Publish error:", err)
		}
	}
}

func (app *application) printMessagesFrom(ctx context.Context, sub *pubsub.Subscription) {
	for {
		m, err := sub.Next(ctx)
		if err != nil {
			panic(err)
		}
		line := string(m.Message.Data)
		if strings.HasPrefix(line, "/") {
			go app.handleCmnds(m.ReceivedFrom.String(), line) //println("got a command: ", line)
			continue
		}
		from := app.store[m.ReceivedFrom.String()]
		fmt.Println(from, ": ", line)
	}
}

func (app *application) handleCmnds(from, cmd string) {
	arr := strings.Split(cmd, " ")
	switch arr[0] {
	case "/name":
		println(from, " : ", arr[1])
	default:
		// log. an error
	}
}
