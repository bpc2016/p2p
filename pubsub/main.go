package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
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

type application struct {
	debug bool
	help  map[string]string
}

var my application

func main() {
	debugF := flag.Bool("d", false, "debug")
	portF := flag.Int("p", 0, "port to use")
	nickF := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	roomF := flag.String("room", "akumuji", "name of chat room to join")

	flag.Parse()
	ctx := context.Background()

	// this app requires internet connectivity
	if !connected() {
		panic("check your internet connection")
	}

	// setup the appllication
	my = application{
		debug: *debugF, // if set, makes app less verbose
		help:  help,    // help called with .help
	}

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
	nick := *nickF
	if len(nick) == 0 {
		nick = defaultNick(h.ID())
	}

	// cr defined here so that we can easily move to another
	cr := ChatRoom{
		ctx:  ctx,
		ps:   ps,
		self: h.ID(),
		nick: nick,
		home: *roomF,
		quit: make(chan struct{}),
	}

	// joining room = *roomF takes care of topic,
	// now includes discovery, hence the h
	if err := cr.JoinChat(h, *roomF); err != nil {
		panic(err)
	}

	cr.homeTopic = cr.topic // keep this one, for use by the `.home` command

	println("You have to be online for this to work!")

	// println("best to start with a nickname: if you havent already, type '.bye' and start again with flag '-nick <yournickname>'")
	gethelp("0")
	// loop that prints responses, user sends message `.bye` to quit
	cr.printMessagesFrom(h) // h so that we can use JoinChat
}

func connected() bool {
	if _, err := http.Get("http://clients3.google.com/generate_204"); err != nil {
		return false
	}
	return true
}

// my own println - replace a verbiage with x
func (my *application) Println(x string, a ...any) (n int, err error) {
	if my.debug {
		return fmt.Println(a...)
	}
	return fmt.Print(x) // just replace with these
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
	topicname := cr.topic.String()
	dutil.Advertise(ctx, routingDiscovery, topicname)

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	for !anyConnected {
		my.Println("\n", "Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, topicname)
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				my.Println("-", "Failed connecting to ", peer.ID.String(), ", error:", err)
			} else {
				my.Println("\n@\n", "Connected to:", peer.ID.String())
				anyConnected = true
			}
		}
	}
	my.Println("\nPeer discovery complete\n\n", "Peer discovery complete")
}

// capture keystrokes and produce messages
// ctx and topic taken care of by chatroom
func (cr *ChatRoom) streamConsoleTo() {
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

func printLine(from, msg string) (n int, err error) {
	// Green console colour: 	\x1b[32m
	// Reset console colour: 	\x1b[0m
	return fmt.Printf("\x1b[32m%s\x1b[0m: %s\n\n", from, msg)
}

// for multiplexed chat usage - use with readloop
func (cr *ChatRoom) printMessagesFrom(h host.Host) {
	ticker := time.NewTicker(5 * time.Second)
OUT:
	for {
		select {
		case cm := <-cr.Messages:
			// fmt.Println(cm.SenderNick, ": ", cm.Message)
			printLine(cm.SenderNick, cm.Message)
		case cc := <-cr.Commands:
			// fmt.Println(cc.SenderNick, ">> ", cc.Cmd, cc.Params)
			if err := cr.HandleRemote(cc, h); err != nil {
				fmt.Printf("handle command error: %v\n", err)
				continue
			}
		case <-ticker.C:
			// do nothing
		case <-cr.quit:
			break OUT
		}
	}
}

// remote function calls all are like ".command"
func (cr *ChatRoom) HandleRemote(cc *ChatCommand, h host.Host) error {
	// typical example: shift room
	switch cc.Cmd {
	case ".join":
		room := cc.Params[0] // already know this is nonempty
		if err := cr.JoinChat(h, room); err != nil {
			panic(err)
		}
		return nil
	case ".peers":
		ids := cr.ListPeers()
		for _, p := range ids {
			fmt.Printf("%v\n", p)
		}
		return nil
	case ".to":
		if !strings.Contains(cc.Params[0], cr.nick) {
			return nil // ignore message
		}
		// fmt.Println(cc.SenderNick, ": ", cc.Params[1])
		printLine(cc.SenderNick, cc.Params[1])
		return nil
	default:
		return errNotFound
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
