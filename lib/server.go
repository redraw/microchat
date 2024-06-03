package lib

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	rooms    map[string]*room
	commands chan command
	members  map[net.Addr]*client
	addr     string
	useTLS   bool
	certFile string
	keyFile  string
	mu       sync.Mutex
}

func NewServer(addr string, useTLS bool, certFile string, keyFile string) *Server {
	return &Server{
		rooms:    make(map[string]*room),
		commands: make(chan command),
		members:  make(map[net.Addr]*client),
		addr:     addr,
		useTLS:   useTLS,
		certFile: certFile,
		keyFile:  keyFile,
	}
}

func (s *Server) newListener() (net.Listener, error) {
	if s.useTLS {
		cert, err := tls.LoadX509KeyPair(s.certFile, s.keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %s", err)
		}

		config := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		return tls.Listen("tcp", s.addr, config)
	}

	return net.Listen("tcp", s.addr)
}

func (s *Server) Run() error {
	listener, err := s.newListener()
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("Server started on %s (tls=%v)", listener.Addr(), s.useTLS)
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

func (s *Server) handler() {
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
			s.listMembers(cmd.client)
		case CMD_WHOIS:
			s.whois(cmd.client, cmd.args)
		case CMD_ME:
			s.me(cmd.client)
		case CMD_HELP:
			s.help(cmd.client)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

func (s *Server) newClient(conn net.Conn) *client {
	log.Printf("new client has joined: %s", conn.RemoteAddr())

	client := &client{
		conn:     conn,
		nick:     conn.RemoteAddr().String(),
		commands: s.commands,
		joined:   time.Now(),
	}

	s.addMember(client)
	s.join(client, []string{"#lobby"})

	return client
}

func (s *Server) addMember(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[c.conn.RemoteAddr()] = c
}

func (s *Server) removeMember(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.room != nil {
		c.room.removeMember(c)
	}
	delete(s.members, c.conn.RemoteAddr())
}

func (s *Server) nick(c *client, args []string) {
	if len(args) < 1 {
		c.msg("nick is required. usage: /nick NAME")
		return
	}

	previousNick := c.nick
	newNick := args[0]

	if s.findClientByNick(newNick) != nil {
		c.msg(fmt.Sprintf("nick %s is already taken", newNick))
		return
	}

	c.nick = newNick
	c.msg(fmt.Sprintf("all right, I will call you %s", c.nick))

	if c.room != nil {
		c.room.broadcast(c, fmt.Sprintf("%s is now known as %s", previousNick, c.nick))
	}
}

func (s *Server) getOrCreateRoom(name string) *room {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !strings.HasPrefix(name, "#") {
		name = fmt.Sprintf("#%s", name)
	}

	r, ok := s.rooms[name]
	if !ok {
		r = &room{
			name:    name,
			members: make(map[net.Addr]*client),
		}
		s.rooms[name] = r
	}
	return r
}

func (s *Server) join(c *client, args []string) {
	if len(args) < 1 {
		c.msg("room name is required. usage: /join ROOM_NAME")
		return
	}
	r := s.getOrCreateRoom(args[0])

	c.join(r)
	c.msg(fmt.Sprintf("welcome to %s", r.name))
}

func (s *Server) listRooms(c *client) {
	var rooms []string
	for name := range s.rooms {
		rooms = append(rooms, name)
	}

	c.msg(fmt.Sprintf("available rooms: %s", strings.Join(rooms, ", ")))
}

func (s *Server) msg(c *client, args []string) {
	msg := strings.Join(args[:], " ")
	if c.room != nil {
		c.room.broadcast(c, c.nick+": "+msg)
		return
	}
	c.msg("you must join a room first.")
}

func (s *Server) msgUser(c *client, args []string) {
	if len(args) < 2 {
		c.msg("message is required, usage: /msg NICK MSG")
		return
	}

	receiver := s.findClientByNick(args[0])
	if receiver == nil {
		c.msg(fmt.Sprintf("user %s not found", args[0]))
		return
	}

	if receiver == c {
		c.msg("you can't send message to yourself")
		return
	}

	msg := strings.Join(args[1:], " ")
	receiver.msg(fmt.Sprintf("private message from %s: %s", c.nick, msg))
}

func (s *Server) findClientByNick(nick string) *client {
	for _, c := range s.members {
		if c.nick == nick {
			return c
		}
	}
	return nil
}

func (s *Server) listMembers(c *client) {
	var members []string
	for _, m := range s.members {
		members = append(members, m.nick)
	}
	c.msg(fmt.Sprintf("members: %s", strings.Join(members, ", ")))
}

func (s *Server) whois(c *client, args []string) {
	if len(args) < 1 {
		c.msg("nick is required. usage: /whois NICK")
		return
	}

	nick := args[0]
	client := s.findClientByNick(nick)
	if client == nil {
		c.msg(fmt.Sprintf("user %s not found", nick))
		return
	}

	c.whois(client)
}

func (s *Server) me(c *client) {
	c.whois(c)
}

func (s *Server) help(c *client) {
	c.msg("available commands:")
	c.msg("/nick NAME - set your nickname")
	c.msg("/join ROOM - join a room")
	c.msg("/rooms - list available rooms")
	c.msg("/msg MSG - send a message to the current room")
	c.msg("/msg USER MSG - send a private message to a user")
	c.msg("/members - list members in the current room")
	c.msg("/whois USER - get information about a user")
	c.msg("/me - get information about yourself")
	c.msg("/help - display this help message")
	c.msg("/quit - leave the chat")
}

func (s *Server) quit(c *client) {
	defer c.conn.Close()

	log.Printf("client has left the chat: %s", c.conn.RemoteAddr().String())
	s.removeMember(c)

	c.msg("sad to see you go =(")
}
