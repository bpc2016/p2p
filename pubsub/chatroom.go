package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// ChatRoomBufSize is the number of incoming messages to buffer for each topic.
const ChatRoomBufSize = 128

// ChatRoom represents a subscription to a single PubSub topic. Messages
// can be published to the topic with ChatRoom.Publish, and received
// messages are pushed to the Messages channel.
type ChatRoom struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *ChatMessage
	// Data is a channel for binary carried as payload
	Data chan *ChatData

	ctx   context.Context
	ps    *pubsub.PubSub
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	roomName  string
	self      peer.ID
	nick      string
	home      string
	homeTopic *pubsub.Topic
	quit      chan struct{}
}

// ChatMessage gets converted to/from JSON and sent in the body of pubsub messages.
type ChatMessage struct {
	Message    string
	To         string
	Payload    []byte
	SenderID   string
	SenderNick string
}

type ChatData struct {
	Data       []byte
	SenderNick string
}

// call this on a chatroom object in main()
func (cr *ChatRoom) JoinChat(h host.Host, roomName string) error {
	var (
		topic *pubsub.Topic
		err   error
	)
	if cr.homeTopic != nil && roomName == cr.home {
		// we are returning home ...
		topic = cr.homeTopic
	} else {
		// join the pubsub topic
		topic, err = cr.ps.Join(topicName(roomName))
		if err != nil {
			return err
		}
	}

	// and subscribe to it
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	cr.topic = topic
	cr.sub = sub
	cr.roomName = roomName
	cr.Messages = make(chan *ChatMessage, ChatRoomBufSize)
	cr.Data = make(chan *ChatData, ChatRoomBufSize)

	// use DHT
	go cr.discoverPeers(cr.ctx, h)

	// write message
	go cr.streamConsoleTo(h)

	// start reading messages from the subscription in a loop
	go cr.readLoop(h)
	return nil
}

// Publish sends a message to the pubsub topic.
func (cr *ChatRoom) Publish(message string, to string, payload []byte) error {
	m := ChatMessage{
		Message:    message,
		To:         to,
		Payload:    payload,
		SenderID:   cr.self.Pretty(),
		SenderNick: cr.nick,
	}

	// fmt.Printf("** message: %s to %q\n", m.Message, m.To) // ***

	msgBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}

func (cr *ChatRoom) ListPeers() []peer.ID {
	return cr.ps.ListPeers(topicName(cr.roomName))
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *ChatRoom) readLoop(h host.Host) {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			close(cr.Messages)
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == cr.self {
			continue
		}

		cm := new(ChatMessage)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}

		// is this personal message, skip if not mine
		minere := regexp.MustCompile(shortID(h.ID()))
		if cm.To != "" && !minere.MatchString(cm.To) {
			continue
		}

		// is this a remote command?
		if strings.HasPrefix(cm.Message, "/") {
			if !cr.validCommand(cm.Message) {
				fmt.Printf("invalid command : %q\n", cm.Message)
				continue
			}
			cm.To = ""
			// prep, if there is a payload, it comes out here as `p`
			p, err := cr.handleCommands(&cm.Message, &cm.To, h)
			if err != nil {
				fmt.Printf("handlecomands error: %v\n", err)
				continue
			}
			// new message back to sender
			if err := cr.Publish(cm.Message, cm.Sender(), p); err != nil {
				fmt.Printf("publish error: %v\n", err)
			}
			continue
		}
		// send the payloaded messages to data channel
		if cm.Payload != nil && string(cm.Payload) != "" {
			data := new(ChatData)
			data.Data = cm.Payload
			data.SenderNick = cm.SenderNick
			// send both
			cr.Messages <- cm
			cr.Data <- data
			continue
		}
		// send valid messages onto the Messages channel
		cr.Messages <- cm
	}
}

// prefix the chatroom name (why?)
func topicName(roomName string) string {
	return "chat-room:" + roomName
}

// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func JoinChatRoom(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, nickname string, roomName string) (*ChatRoom, error) {
	// join the pubsub topic
	topic, err := ps.Join(topicName(roomName))
	if err != nil {
		return nil, err
	}

	// and subscribe to it
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	cr := &ChatRoom{
		ctx:      ctx,
		ps:       ps,
		topic:    topic,
		sub:      sub,
		self:     selfID,
		nick:     nickname,
		roomName: roomName,
		Messages: make(chan *ChatMessage, ChatRoomBufSize),
	}

	return cr, nil
}
