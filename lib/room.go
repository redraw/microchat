package lib

import (
	"fmt"
	"net"
	"sync"
)

type room struct {
	name    string
	members map[net.Addr]*client
	mu      sync.Mutex
}

func (r *room) broadcast(sender *client, msg string) {
	for addr, m := range r.members {
		if sender.conn.RemoteAddr() != addr {
			m.msg(msg)
		}
	}
}

func (r *room) addMember(c *client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[c.conn.RemoteAddr()] = c
	r.broadcast(c, fmt.Sprintf("%s has joined the room", c.nick))
}

func (r *room) removeMember(c *client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldRoom := r
	delete(r.members, c.conn.RemoteAddr())
	oldRoom.broadcast(c, fmt.Sprintf("%s has left the room", c.nick))
}
