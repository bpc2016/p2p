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
	case "/fetch", "/check", "/to", "/who", "/iam", "/q", "/h", "/in?", "/inf", "/peers",
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

	switch cmd {
	case "/who": // /to all /iam - query who is there
		inject(cr, "/iam", "", "")
		return nil, errSkip
	case "/inf": // given as '/in <user>'
		user := pars
		inject(cr, "/fetch", user, "this is fixed content")
		return nil, errSkip
	case "/fetch": // fetch <addr>, usually called as /to <user> /fetch <addr>
		// return this as a byte slice, manipulated
		payload = sampleFetch(pars)
		*s = "check the json payload\n"
	case "/to": // formmat /to <addr> message
		readdr := regexp.MustCompile(`(\S+)\s*(.*)`)
		reall := regexp.MustCompile(`^all$`)
		// the single address follows directly, match this with `readLoop`
		*to = readdr.ReplaceAllString(pars, "$1")
		*to = reall.ReplaceAllString(*to, "") // fancy way of saying all --> ''
		*s = fmt.Sprintf("%s\n", readdr.ReplaceAllString(pars, "$2"))
	case "/peers": // example of a `local command`: never published
		//            purely for information to the user
		for _, p := range cr.ListPeers() {
			fmt.Printf("%v\n", p)
		}
		return nil, errSkip
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

// a wrapper for RPCs
// we use /to <addr> /fetch <stuff> on the command line
// instead, we have /inj <addr> with preset stuff
func inject(cr *ChatRoom, cmd, to, content string) {
	str := fmt.Sprintf("%s %s\n", cmd, content)
	cr.Publish(str, to, []byte{})
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
	return bytes
}

// return the short form of the senderID
func (cm *ChatMessage) Sender() (sender string) {
	sender = cm.SenderID
	sender = sender[len(sender)-8:]
	return
}
