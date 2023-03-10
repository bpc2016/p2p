package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

func main() {
	portF := flag.Int("p", 0, "port to use")
	// parse some flags to set our nickname and the room to join
	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	roomFlag := flag.String("room", "akumuji", "name of chat room to join")

	flag.Parse()
	ctx := context.Background()

	listener := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *portF)
	h, err := libp2p.New(libp2p.ListenAddrStrings(listener))
	if err != nil {
		panic(err)
	}

	// subscription is the 1st thing: done by the host
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	// use the nickname from the cli flag, or a default if blank
	nick := *nickFlag
	if len(nick) == 0 {
		nick = defaultNick(h.ID())
	}

	// join the room from the cli flag, or the flag default
	room := *roomFlag

	// join the chat room - takes care of topic <--> room
	cr, err := JoinChatRoom(ctx, ps, h.ID(), nick, room)
	if err != nil {
		panic(err)
	}

	// use DHT
	go cr.discoverPeers(ctx, h)

	// groutine that sends out messages
	go cr.streamConsoleTo() // ctx, cr.topic)

	// // we want to keep looking at attached hosts
	// go func() {
	// 	for {
	// 		time.Sleep(1 * time.Minute) // use a ticker inside a select
	// 		println("--------------------------")
	// 		// log.Println("mystore:")
	// 		// for p := range myStore {
	// 		// 	println(p) // log.Printf("id: %s\n", p)
	// 		// }
	// 		println("--------------------------")
	// 	}
	// }()

	// loop that prints responses
	cr.printMessagesFrom() //ctx, cr.sub)
}

// called by discoverPeers
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

// use topic from the chatroot, could have also been set as a parameter
func (cr *ChatRoom) discoverPeers(ctx context.Context, h host.Host) {
	kademliaDHT := initDHT(ctx, h)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	// dutil.Advertise(ctx, routingDiscovery, *topicNameF)
	dutil.Advertise(ctx, routingDiscovery, cr.topic.String())

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	for !anyConnected {
		fmt.Println("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, cr.topic.String()) // *topicNameF
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
				anyConnected = true
			}
		}
	}
	fmt.Println("Peer discovery complete")

	// continue looking ...
	go cr.fetchMore(ctx, routingDiscovery, h)
}

// long term search for peers
func (cr *ChatRoom) fetchMore(ctx context.Context, routingDiscovery *drouting.RoutingDiscovery, h host.Host) {
	for {
		//fmt.Println("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, cr.topic.String()) // *topicNameF
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			// peerAD := peer.ID.String()
			// test connectivity
			// err := h.Connect(ctx, peer)
			h.Connect(ctx, peer) // test connected
			/*
				if err != nil {
					if app.store[peerAD] != "" {
						delete(app.store, peerAD) // clear departed
					}
				} else {
					// silently add new found
					app.found <- peer
				}
			*/
		}
	}
}

func (cr *ChatRoom) streamConsoleTo() { //ctx context.Context, topic *pubsub.Topic) {
	reader := bufio.NewReader(os.Stdin)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if err := cr.Publish(s); err != nil {
			fmt.Println("### Publish error:", err)
		}
	}
}

func (cr *ChatRoom) printMessagesFrom() { //ctx context.Context, sub *pubsub.Subscription) {
	for {
		m, err := cr.sub.Next(cr.ctx)
		if err != nil {
			panic(err)
		}
		// line := string(m.Message.Data)

		cm := new(ChatMessage)
		err = json.Unmarshal(m.Message.Data, cm)
		if err != nil {
			continue
		}
		if strings.HasPrefix(cm.Message, "/") {
			go handleCmnds(cr.nick, cm.Message) //println("got a command: ", line)
			continue
		}
		// from := app.store[m.ReceivedFrom.String()]
		// if from == "" {
		// 	from = "me"
		// }
		fmt.Println(cr.nick, ": ", cm.Message) // use nick
	}
}

func handleCmnds(from, cmd string) {
	arr := strings.Split(cmd, " ")
	switch arr[0] {
	case "/name":
		println(from, " : ", arr[1])
	default:
		// log. an error
	}
}

// defaultNick generates a nickname based on the $USER environment variable and
// the last 8 chars of a peer ID.
func defaultNick(p peer.ID) string {
	return fmt.Sprintf("%s-%s", os.Getenv("USER"), shortID(p))
}

// shortID returns the last 8 chars of a base58-encoded peer id.
func shortID(p peer.ID) string {
	pretty := p.Pretty()
	return pretty[len(pretty)-8:]
}
