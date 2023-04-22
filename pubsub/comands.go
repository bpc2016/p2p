package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type ChatCommand struct {
	Cmd        string
	SenderID   string
	SenderNick string
	Params     []string
	Payload    []byte
}

// move to commands.go
func (cc *ChatCommand) ParseCommand(cm *ChatMessage) error {
	cc.SenderID = cm.SenderID
	cc.SenderNick = cm.SenderNick
	cc.Payload = cm.Payload
	str := strings.TrimSuffix(cm.Message, "\n")
	arr := strings.Split(string(str), " ")
	cc.Cmd = arr[0]
	rest := strings.Join(arr[1:], " ")
	// check if the command exist, right length : return error else
	return cc.Find(rest)
}

// return the short form of the senderID
func (cc *ChatCommand) Sender() (sender string) {
	sender = cc.SenderID
	sender = sender[len(sender)-8:]
	return
}

func (cr *ChatRoom) ParseLocalCommand(cd *string, prs *[]string, cm *ChatMessage) error {
	str := strings.TrimSuffix(cm.Message, "\n")
	arr := strings.Split(string(str), " ")
	*cd = arr[0]
	rest := strings.Join(arr[1:], " ")
	return cr.find(*cd, prs, rest)
}

var (
	errNotFound  = errors.New("command not found")
	errBadSyntax = errors.New("bad command syntax")
)

// RPCs all prefixed with '.'
func (cc *ChatCommand) Find(rest string) error {
	switch cc.Cmd {
	case ".join", ".fetch":
		str := strings.TrimSpace(rest)
		cc.Params = strings.Split(str, " ") // []string{str}
		return nil
	case ".peers", ".who":
		return nil
	default:
		return errNotFound
	}
}

// remote function calls all are like ".command"
func (cr *ChatRoom) HandleRemote(cc *ChatCommand, h host.Host) error {
	// typical example: shift room
	switch cc.Cmd {
	case ".join":
		if cc.Params[0] == "" {
			return fmt.Errorf(".join empty room")
		}
		if len(cc.Params) > 1 {
			return fmt.Errorf(".join bad parameters")
		}
		room := cc.Params[0] // already know this is nonempty
		if err := cr.JoinChat(h, room); err != nil {
			return err
		}
		return nil
	case ".peers":
		ids := cr.ListPeers()
		for _, p := range ids {
			fmt.Printf("%v\n", p)
		}
		return nil
	case ".who":
		// force all to reveal themselves, and send the results to me
		iam := cr.nick + " " + shortID(h.ID()) + "\n"
		// respond with a message back to sender
		//println("this was from: ", sender) // ****
		if err := cr.Publish(iam, cc.Sender()); err != nil {
			return err
		}
		return nil
	case ".fetch": // .fetch addr wdata - where the wdata comes from an address

		fmt.Println("parameters: ", cc.Params) // ****
		// return fmt.Errorf(".fetch fails with missing data : %v", cc)
		to := cc.Params[0]   // who this message is going to
		addr := cc.Params[1] // the desired ftree data point
		if to == "" || addr == "" {
			return fmt.Errorf(".fetch fails with missing data to: %q, addr=%q", to, addr)
		}
		// fetch the desired data
		data := []byte("desired data")
		// return to sender
		if err := cr.Publish("fetched for you ...\n", cc.Sender(), data); err != nil {
			return fmt.Errorf("publish error: %v", err)
		}
		return nil
	default:
		return errNotFound
	}
}

// local commands all prefixed with '/'
// verify the cmd exists and syntax is right, use string `rest` to fill `prs` if nec
func (cr *ChatRoom) find(cmd string, prs *[]string, rest string) error {

	fmt.Printf("enter `find` with rest: %q\n", rest)

	switch cmd {
	case "/join", "/help", "/h", "/test", "/request", "/fetch":
		str := strings.TrimSpace(rest)
		*prs = strings.Split(str, " ") // []string{str}

		// join?
		if cmd == "/join" {
			// no empty rooms
			if str == "" || strings.Contains(str, " ") {
				return errBadSyntax
			}
			// check we are not already there
			if topicName(str) == cr.topic.String() {
				return errors.New("already at this topic")
			}
		}
		fmt.Println("exit `find` ok")
		return nil
	case "/home", "/peers", "/quit", "/q":
		return nil
	default:
		return errNotFound
	}
}

func (cr *ChatRoom) handleLocal(s string, h host.Host) {

	// wrap the message up
	cm := &ChatMessage{Message: s} //new(ChatMessage)

	// identify & parse
	cmd, prs := "", []string{}
	if err := cr.ParseLocalCommand(&cmd, &prs, cm); err != nil {
		fmt.Printf("parselocal error: %v\n", err)
		return
	}

	// process cmds
	switch cmd {
	case "/join":
		room := prs[0] // already know this is nonempty
		if err := cr.JoinChat(h, room); err != nil {
			fmt.Printf("join chat error: %v\n", err)
		}
	case "/home": // return to base
		if err := cr.JoinChat(h, cr.home); err != nil {
			fmt.Printf("join chat error: %v\n", err)
		}
	case "/peers": // list peers
		ids := cr.ListPeers()
		for _, p := range ids {
			fmt.Printf("%v\n", p)
		}
	case "/quit", "/q": // exit program
		cr.quit <- struct{}{}
	case "/help", "/h":
		it := prs[0]
		gethelp(it)
	case "/request": // /request <ftree-addr>
		addr := prs[0]
		// outputs
		fmt.Printf("this is a random consequence of the request for: %s\n", addr)
	case "/fetch": // fetch <addr>
		from := prs[0]
		cr.Publish("/request tttgh\n", from)
	case "/test": // test <addr> <about> sends a private message there
		to := prs[0]
		// about := prs[1]
		// cr.Publish(".who\n", to, []byte(about))
		cr.Publish("/peers\n", to)
	}
}

// local commands look like `/join another_topic`
func (cr *ChatRoom) HandleLocal(msg *pubsub.Message, h host.Host) {
	cm := new(ChatMessage)
	err := json.Unmarshal(msg.Data, cm)
	if err != nil {
		return
	}
	// really only want this for commands like `.join others``
	if !strings.HasPrefix(cm.Message, "/") {
		return
	}
	// verify this command exits, right syntax
	cmd, prs := "", []string{}
	if err := cr.ParseLocalCommand(&cmd, &prs, cm); err != nil {
		return
	}

	switch cmd {
	case "/join":
		room := prs[0] // already know this is nonempty
		if err := cr.JoinChat(h, room); err != nil {
			panic(err)
		}
	case "/home": // return to base
		if err := cr.Publish(fmt.Sprintf("/join %s", cr.home), ""); err != nil {
			return
		}
	case "/peers": // list peers
		ids := cr.ListPeers()
		for _, p := range ids {
			fmt.Printf("%v\n", p)
		}
	case "/quit", "/q": // exit program
		cr.quit <- struct{}{}
	case "/help", "/h":
		it := prs[0]
		gethelp(it)
	case "/test": // test <addr> <about> sends a private message there
		to := prs[0]
		// about := prs[1]
		// cr.Publish(".who\n", to, []byte(about))
		cr.Publish("/peers\n", to)
	}
}

func Private(msg string, to *string) error {
	parts := strings.Split(msg, " ")
	switch parts[0] {
	case ".fetch":
		if len(parts) < 2 {
			return fmt.Errorf(".fetch has too few parameters")
		}
		*to = parts[1]
	}
	return nil
}
