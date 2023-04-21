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
}

// move to commands.go
func (cc *ChatCommand) ParseCommand(cm *ChatMessage) error {
	cc.SenderID = cm.SenderID
	cc.SenderNick = cm.SenderNick
	str := strings.TrimSuffix(cm.Message, "\n")
	arr := strings.Split(string(str), " ")
	cc.Cmd = arr[0]
	rest := strings.Join(arr[1:], " ")
	// check if the command exist, right length : return error else
	return cc.Find(rest)
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

// RPCs all prefixed with '/'
func (cc *ChatCommand) Find(rest string) error {
	switch cc.Cmd {
	case ".join":
		str := strings.TrimSpace(rest)
		if str == "" || strings.Contains(str, " ") {
			return errBadSyntax
		}
		cc.Params = []string{str}
		return nil
	case ".peers":
		return nil
	case ".to":
		if !strings.Contains(rest, ":") {
			return errBadSyntax
		}
		cc.Params = strings.Split(rest, ":")
		if len(cc.Params) != 2 {
			return errBadSyntax
		}
		cc.Params[1] = strings.TrimSpace(cc.Params[1])
		return nil
	default:
		return errNotFound
	}
}

// local commands all prefixed with '/'
// verify the cmd exists and syntax is right, use string `rest` to fill `prs` if nec
func (cr *ChatRoom) find(cmd string, prs *[]string, rest string) error {
	switch cmd {
	case "/join", "/help", "/h":
		str := strings.TrimSpace(rest)
		*prs = []string{str}

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
		return nil
	case "/home", "/peers", "/quit", "/q":
		return nil
	default:
		return errNotFound
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
		if err := cr.Publish(fmt.Sprintf("/join %s", cr.home)); err != nil {
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
		// fmt.Printf("asked for topic %s\n", it)
	}
}
