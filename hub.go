package luna

import (
	"encoding/json"
	"fmt"
	"github.com/olahol/melody"
	"sync"
	"time"
)

//Hub holds all channels in a slice. Clients can subscribe and send message to channels through the Hub
type Hub struct {
	Channels []*Channel //Slice of channels in this Hub
}

//Subscribe subscribes a client to a channel. Create channel if it does not exists before
func (h *Hub) Subscribe(id string, session *melody.Session) *Channel {

	ch := h.GetChannel(id)
	if ch == nil { //create channel if this is the first time or if channel does not not exists
		ch = &Channel{}
		ch.Id = id
		ch.Clients = make(map[*Client]bool)

		h.Channels = append(h.Channels, ch)
	}

	ch.Subscribe(session)
	return ch
}

func (h *Hub) UnSubscribe(id string, session *melody.Session) {

	ch := h.GetChannel(id)
	if ch != nil {
		ch.UnSubscribe(session)
	}
}

//GetChannel returns a Channel identified by @param id
func (h *Hub) GetChannel(id string) *Channel {

	for _, ch := range h.Channels {

		if ch != nil && ch.Id == id {
			return ch
		}
	}

	return nil
}

//Send sends data payload to a channel. This broadcast @params data to all connected clients
func (h *Hub) Send(channel string, data interface{}) {

	ch := h.GetChannel(channel)
	if ch != nil {
		ch.Send(data)
	}
}

//EnsureClean keep clients slice clean. Remove all clients that has been idle for more than 10minutes
func (h *Hub) EnsureClean() {

	for {
		time.Sleep(10 * time.Minute) //Run every 10min

		for _, ch := range h.Channels {

			for v := range ch.Clients {

				if v != nil {
					if t := time.Now().Sub(v.LastSeen); t.Minutes() >= 10 && v.Session.IsClosed() {

						delete(ch.Clients, v)
					}
				}
			}
		}
	}
}

//Count returns number of connected channels
func (h *Hub) Count() int {
	return len(h.Channels)
}

//ClientsCount returns no of connected clients in a channel
func (h *Hub) ClientsCount(id string) int {

	ch := h.GetChannel(id)
	if ch != nil {
		return len(ch.Clients)
	}

	return 0
}

type Client struct {
	Session  *melody.Session
	LastSeen time.Time
}

//A message channel. Multiple clients can subscribe to a channel, message will be broadcast to all clients
//anytime Channel.Send(data) is called
type Channel struct {
	Id      string
	Mtx     sync.Mutex
	Clients map[*Client]bool
}

func (ch *Channel) Lock() {
	ch.Mtx.Lock()
}

func (ch *Channel) UnLock() {
	ch.Mtx.Unlock()
}

//Subscribe subscribes a session to a client.
func (ch *Channel) Subscribe(session *melody.Session) {

	ch.Lock()
	defer ch.UnLock()

	client := &Client{}
	client.Session = session
	client.LastSeen = time.Now()
	ch.Clients[client] = true
}

//UnSubscribe removes a session from a channel
func (ch *Channel) UnSubscribe(session *melody.Session) {

	ch.Lock()
	defer ch.UnLock()

	for v := range ch.Clients {

		if v.Session == session {
			delete(ch.Clients, v)
		}
	}
}

//Send broadcast message to all connected clients and update their last activity time
func (ch *Channel) Send(data interface{}) {

	value, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error while mashalling JSON ", err)
		return
	}

	ch.Lock()
	defer ch.UnLock()
	for v := range ch.Clients {

		if v != nil && !v.Session.IsClosed() {

			v.Session.Write(value)
			v.LastSeen = time.Now()
		}
	}
}
