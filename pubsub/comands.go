package main

import (
	"errors"
	"strings"
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

// RPCs
func (cc *ChatCommand) Find(rest string) error {
	switch cc.Cmd {
	case "/join":
		str := strings.TrimSpace(rest)
		if str == "" || strings.Contains(str, " ") {
			return errBadSyntax
		}
		cc.Params = []string{str}
		return nil
	case "/peers":
		return nil
	case "/to":
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

// local commands all prefixed with '.'
// verify the cmd exists and syntax is right, use string `rest` to fill `prs` if nec
func (cr *ChatRoom) find(cmd string, prs *[]string, rest string) error {
	switch cmd {
	case ".join":
		str := strings.TrimSpace(rest)
		if str == "" || strings.Contains(str, " ") {
			return errBadSyntax
		}
		// check we are not already there
		if topicName(str) == cr.topic.String() {
			return errors.New("already at this topic")
		}
		*prs = []string{str}
		return nil
	case ".home":
		return nil
	case ".peers":
		return nil
	case ".bye":
		return nil
	default:
		return errNotFound
	}
}
