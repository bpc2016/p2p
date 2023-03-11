package main

import (
	"context"
	"encoding/json"
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
	// Commands is a channel for commands issued these are produced from messages - see commands.go
	Commands chan *ChatCommand

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
	SenderID   string
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

	// fmt.Printf("--- I am %s and joining topic %s\n", cr.nick, topic.String())

	cr.topic = topic
	cr.sub = sub
	cr.roomName = roomName
	cr.Messages = make(chan *ChatMessage, ChatRoomBufSize)
	cr.Commands = make(chan *ChatCommand, ChatRoomBufSize)

	// use DHT
	go cr.discoverPeers(cr.ctx, h)

	// write message
	go cr.streamConsoleTo()

	// start reading messages from the subscription in a loop
	go cr.readLoop(h)
	return nil
}

// Publish sends a message to the pubsub topic.
func (cr *ChatRoom) Publish(message string) error {
	m := ChatMessage{
		Message:    message,
		SenderID:   cr.self.Pretty(),
		SenderNick: cr.nick,
	}
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
			println()
			cr.HandleLocal(msg, h)
			continue
		}
		cm := new(ChatMessage)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// is this a local command?
		if strings.HasPrefix(cm.Message, ".") {
			continue
		}
		// is this a remote command?
		if strings.HasPrefix(cm.Message, "/") {
			cc := new(ChatCommand)
			if err := cc.ParseCommand(cm); err != nil {
				continue
			}
			// send valid comand
			cr.Commands <- cc
			continue
		}
		// send valid messages onto the Messages channel
		cr.Messages <- cm
	}
}

func topicName(roomName string) string {
	return "chat-room:" + roomName
}

// ------------------ old ---------------------
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

	// start reading messages from the subscription in a loop
	// go cr.readLoop()
	return cr, nil
}
