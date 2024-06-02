package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

type server struct {
	rooms    map[string]*room
	commands chan command
	members  map[net.Addr]*client
	mu       sync.RWMutex
}

func newServer() *server {
	return &server{
		rooms:    make(map[string]*room),
		commands: make(chan command),
		members:  make(map[net.Addr]*client),
	}
}

func (s *server) listen(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("server started on %s", listener.Addr())
	go s.handler()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %s", err.Error())
			continue
		}
		c := s.newClient(conn)
		go c.readInput()
	}
}

func (s *server) handler() {
	for cmd := range s.commands {
		switch cmd.id {
		case CMD_NICK:
			s.nick(cmd.client, cmd.args)
		case CMD_JOIN:
			s.join(cmd.client, cmd.args)
		case CMD_ROOMS:
			s.listRooms(cmd.client)
		case CMD_MSG:
			s.msg(cmd.client, cmd.args)
		case CMD_MSG_USER:
			s.msgUser(cmd.client, cmd.args)
		case CMD_LIST_MEMBERS:
			s.listMembers(cmd.client, cmd.args)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

func (s *server) newClient(conn net.Conn) *client {
	log.Printf("new client has joined: %s", conn.RemoteAddr())

	client := &client{
		conn:     conn,
		nick:     conn.RemoteAddr().String(),
		commands: s.commands,
	}

	s.addMember(client)
	s.join(client, []string{"/join", "#lobby"})

	return client
}

func (s *server) addMember(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[c.conn.RemoteAddr()] = c
}

func (s *server) removeMember(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.members, c.conn.RemoteAddr())
}

func (s *server) nick(c *client, args []string) {
	if len(args) < 2 {
		c.msg("nick is required. usage: /nick NAME")
		return
	}

	previousNick := c.nick
	newNick := args[1]

	if s.findClientByNick(newNick) != nil {
		c.msg(fmt.Sprintf("nick %s is already taken", newNick))
		return
	}

	c.nick = newNick
	c.msg(fmt.Sprintf("all right, I will call you %s", c.nick))
	c.room.broadcast(c, fmt.Sprintf("%s is now known as %s", previousNick, c.nick))
}

func (s *server) join(c *client, args []string) {
	if len(args) < 2 {
		c.msg("room name is required. usage: /join ROOM_NAME")
		return
	}

	roomName := args[1]

	if len(roomName) > 1 && roomName[0] != '#' {
		roomName = fmt.Sprintf("#%s", roomName)
	}

	r, ok := s.rooms[roomName]
	if !ok {
		r = &room{
			name:    roomName,
			members: make(map[net.Addr]*client),
		}
		s.rooms[roomName] = r
	}

	s.addRoomMember(r, c)
	r.broadcast(c, fmt.Sprintf("%s joined the room", c.nick))
	c.msg(fmt.Sprintf("welcome to %s", roomName))
}

func (s *server) addRoomMember(r *room, c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.members[c.conn.RemoteAddr()] = c
	s.quitCurrentRoom(c)
	c.room = r
}

func (s *server) listRooms(c *client) {
	var rooms []string
	for name := range s.rooms {
		rooms = append(rooms, name)
	}

	c.msg(fmt.Sprintf("available rooms: %s", strings.Join(rooms, ", ")))
}

func (s *server) msg(c *client, args []string) {
	msg := strings.Join(args[:], " ")
	c.room.broadcast(c, c.nick+": "+msg)
}

func (s *server) msgUser(c *client, args []string) {
	if len(args) < 3 {
		c.msg("message is required, usage: /msg NICK MSG")
		return
	}

	receiver := s.findClientByNick(args[1])
	if receiver == nil {
		c.msg(fmt.Sprintf("user %s not found", args[1]))
		return
	}

	if receiver == c {
		c.msg("you can't send message to yourself")
		return
	}

	msg := strings.Join(args[1:], " ")
	receiver.msg(fmt.Sprintf("private message from %s: %s", c.nick, msg))
}

func (s *server) findClientByNick(nick string) *client {
	for _, c := range s.members {
		if c.nick == nick {
			return c
		}
	}
	return nil
}

func (s *server) listMembers(c *client, _ []string) {
	type roomInfo struct {
		name    string
		members []string
	}
	var rooms []roomInfo

	for name, r := range s.rooms {
		var members []string
		for _, m := range r.members {
			members = append(members, m.nick)
		}
		rooms = append(rooms, roomInfo{name, members})
	}

	for _, r := range rooms {
		c.msg(fmt.Sprintf("room: %s, members: %s", r.name, strings.Join(r.members, ", ")))
	}
}

func (s *server) quitCurrentRoom(c *client) {
	if c.room != nil {
		oldRoom := s.rooms[c.room.name]
		delete(s.rooms[c.room.name].members, c.conn.RemoteAddr())
		oldRoom.broadcast(c, fmt.Sprintf("%s has left the room", c.nick))
	}
}

func (s *server) quit(c *client) {
	defer c.conn.Close()

	log.Printf("client has left the chat: %s", c.conn.RemoteAddr().String())
	s.quitCurrentRoom(c)
	s.removeMember(c)

	c.msg("sad to see you go =(")
}
