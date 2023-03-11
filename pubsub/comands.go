package main

import (
	"encoding/json"
	"errors"
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
func ParseLocalCommand(cd *string, prs *[]string, cm *ChatMessage) error {
	str := strings.TrimSuffix(cm.Message, "\n")
	arr := strings.Split(string(str), " ")
	*cd = arr[0]
	rest := strings.Join(arr[1:], " ")
	return find(*cd, prs, rest)
}

var (
	errNotFound  = errors.New("command not found")
	errBadSyntax = errors.New("bad command syntax")
)

// RPCs
func (cc *ChatCommand) Find(rest string) error {
	switch cc.Cmd {
	case "/join":
		str := strings.TrimSpace(rest)
		if str == "" || strings.Contains(str, " ") {
			return errBadSyntax
		}
		cc.Params = []string{str}
	case "/listpeers":
		return nil
	case "/to":
		if !strings.Contains(rest, ":") {
			return errBadSyntax
		}
		arr := strings.Split(rest, ":")
		pars := strings.Split(arr[0], " ") // the target peers
		cc.Params = append(pars, arr[1])   // the message is last item
	default:
		return errNotFound
	}
	return nil
}

// local commands all prefixed with '.'
func find(cmd string, prs *[]string, rest string) error {
	switch cmd {
	case ".join":
		str := strings.TrimSpace(rest)
		if str == "" || strings.Contains(str, " ") {
			return errBadSyntax
		}
		*prs = []string{str}
	}
	return nil
}

// local commands look like `.join another_topic`
func (cr *ChatRoom) HandleLocal(msg *pubsub.Message, h host.Host) {
	cm := new(ChatMessage)
	err := json.Unmarshal(msg.Data, cm)
	if err != nil {
		return
	}
	// really only want this for commands like `.join others``
	if !strings.HasPrefix(cm.Message, ".") {
		return
	}
	// verify this command exits, right syntax
	cmd, prs := "", []string{}
	if err := ParseLocalCommand(&cmd, &prs, cm); err != nil {
		return
	}

	switch cmd {
	case ".join":
		room := prs[0] // already know this is nonempty
		if err := cr.JoinChat(h, room); err != nil {
			panic(err)
		}
	}
}
