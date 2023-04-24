package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
)

var (
	errSkip = errors.New("skip this input")
)

// these are commands that appear at the target, see where they are checked
func (cr *ChatRoom) validCommand(s string) bool {
	re := regexp.MustCompile(`(\S+)\s*(.*)\n`)
	cmd := re.ReplaceAllString(s, "$1")
	switch cmd {
	case "/fetch", "/check", "/to", "/who", "/iam", "/q", "/h", "/in",
		"/iiam":
		return true
	}
	return false
}

// prepare data for pulishing
func (cr *ChatRoom) handleCommands(s, to *string, h host.Host) ([]byte, error) {
	re := regexp.MustCompile(`(\S+)\s*(.*)\n`)
	cmd := re.ReplaceAllString(*s, "$1")
	pars := re.ReplaceAllString(*s, "$2")
	payload := []byte{}
	// fmt.Printf("command: %q, parameters: %q\n", cmd, pars) // *****

	switch cmd {
	case "/in": // given as '/in <user>'
		user := pars
		cr.inject(user)
		return nil, errSkip
	case "/fetch": // fetch <addr>, usually called as /to <user> /fetch <addr>
		addr := pars
		// return this as a byte slice
		payload = sampleFetch(addr)
		*s = "check the json payload\n"
	case "/to": // formmat /to <addr> message
		readdr := regexp.MustCompile(`(\S+)\s*(.*)`)
		// the single address follows directly, match this with `readLoop`
		*to = readdr.ReplaceAllString(pars, "$1")
		if *to == "all" {
			*to = ""
		}
		*s = fmt.Sprintf("%s\n", readdr.ReplaceAllString(pars, "$2"))
	case "/peers": // example of a `local command`: never published
		//            purely for information to the user
		for _, p := range cr.ListPeers() {
			fmt.Printf("%v\n", p)
		}
		return nil, errSkip
	case "/who": // who is available: give me short IDs
		*s = "/to all /iam\n"
	case "/iam": // declare my short ID
		*s = fmt.Sprintf("%s = %s\n", cr.nick, shortID(h.ID()))
	case "/quit", "/q":
		cr.quit <- struct{}{}
	case "/help", "/h":
		gethelp(pars)
		return nil, errSkip
	default:
		return nil, fmt.Errorf("unknown command: %q", cmd)
	}
	return payload, nil
}

// a wraper for sampleFetch
// we use /to <addr> /fetch <stuff> on the command line
// instead, we have /inj <addr> with preset stuff
func (cr *ChatRoom) inject(to string) {
	cr.Publish("/fetch that stuff produced elsewhere\n", to, []byte{})
}

// typical, json encode the payload
func sampleFetch(addr string) []byte {
	type mine = struct {
		Tag  string
		Data string
	}
	obj := mine{
		Tag:  addr,
		Data: strings.ToUpper(addr),
	}
	bytes, _ := json.Marshal(obj) // ignore errors

	// fmt.Printf("out: %v\n", bytes) // ****

	return bytes
}

// return the short form of the senderID
func (cm *ChatMessage) Sender() (sender string) {
	sender = cm.SenderID
	sender = sender[len(sender)-8:]
	return
}

//--------------------- togo ---------------------------

// type ChatCommand struct {
// 	Cmd        string
// 	SenderID   string
// 	SenderNick string
// 	Params     []string
// 	Payload    []byte
// }

// return the short form of the senderID
// func (cc *ChatCommand) Sender() (sender string) {
// 	sender = cc.SenderID
// 	sender = sender[len(sender)-8:]
// 	return
// }

/*
// now used with remote calls
func (cr *ChatRoom) WasparseCommand(cc *ChatCommand, cm *ChatMessage, h host.Host) error {
	cc.SenderID = cm.SenderID
	cc.SenderNick = cm.SenderNick
	cc.Payload = cm.Payload
	// str := strings.TrimSuffix(cm.Message, "\n")
	// arr := strings.Split(string(str), " ")
	re := regexp.MustCompile(`(\S+)\s*(.*)`)
	cc.Cmd = re.ReplaceAllString(cm.Message, "$1")
	// rest := strings.Join(arr[1:], " ")
	// check if the command exist, right length : return error else
	s := cm.Message // join as was originally presented
	// err := cr.handleCommands(&s, &cm.To, nil)
	// if err == errSkip { // only passed when the command is purely local
	// 	return nil
	// }
	// return err
	return cr.handleCommands(&s, &cm.To, h)
}

// previous version
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
*/

// func (cr *ChatRoom) ParseLocalCommand(cd *string, prs *[]string, cm *ChatMessage) error {
// 	str := strings.TrimSuffix(cm.Message, "\n")
// 	arr := strings.Split(string(str), " ")
// 	*cd = arr[0]
// 	rest := strings.Join(arr[1:], " ")
// 	return cr.find(*cd, prs, rest)
// }

/*
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
*/

/*
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
*/

/*
// local commands all prefixed with '/'
// verify the cmd exists and syntax is right, use string `rest` to fill `prs` if nec
func (cr *ChatRoom) find(cmd string, prs *[]string, rest string) error {

	fmt.Printf("enter ONCE `find` with rest: %q\n", rest)

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

*/

/*
func (cr *ChatRoom) OldhandleLocal(s string, h host.Host) {

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
*/

/*
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
*/

/*
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

*/
