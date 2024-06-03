package lib

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type client struct {
	conn     net.Conn
	nick     string
	room     *room
	commands chan<- command
	joined   time.Time
}

const (
	MAX_MSG_BODY = 4096
)

func readLimitedMessage(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	if len(line) > MAX_MSG_BODY {
		line = line[:MAX_MSG_BODY]
	}

	return line, nil
}

func (c *client) readInput() {
	defer c.quit()
	reader := bufio.NewReader(c.conn)

	for {
		msg, err := readLimitedMessage(reader)
		if err != nil {
			log.Printf("failed to read data: %s", err.Error())
			return
		}

		msg = strings.Trim(msg, "\r\n")

		args := strings.Split(msg, " ")
		cmd := strings.TrimSpace(args[0])

		switch {
		case len(cmd) > 0 && cmd[0] == '/':
			switch cmd {
			case "/nick":
				c.commands <- command{
					id:     CMD_NICK,
					client: c,
					args:   args,
				}
			case "/join":
				c.commands <- command{
					id:     CMD_JOIN,
					client: c,
					args:   args,
				}
			case "/rooms":
				c.commands <- command{
					id:     CMD_ROOMS,
					client: c,
				}
			case "/msg":
				c.commands <- command{
					id:     CMD_MSG_USER,
					client: c,
					args:   args,
				}
			case "/members":
				c.commands <- command{
					id:     CMD_LIST_MEMBERS,
					client: c,
				}
			case "/whois":
				c.commands <- command{
					id:     CMD_WHOIS,
					client: c,
					args:   args,
				}
			case "/me":
				c.commands <- command{
					id:     CMD_ME,
					client: c,
				}
			case "/help":
				c.commands <- command{
					id:     CMD_HELP,
					client: c,
				}
			case "/quit":
				return
			default:
				c.err(fmt.Errorf("unknown command: %s", cmd))
			}
		default:
			c.commands <- command{
				id:     CMD_MSG,
				client: c,
				args:   args,
			}
		}
	}
}

func (c *client) err(err error) {
	c.conn.Write([]byte("err: " + err.Error()))
}

func (c *client) msg(msg string) {
	c.conn.Write([]byte(msg + "\n"))
}

func (c *client) whois(other *client) {
	c.msg(fmt.Sprintf("nick: %s", other.nick))
	c.msg(fmt.Sprintf("addr: %s", other.conn.RemoteAddr()))
	c.msg(fmt.Sprintf("joined: %s", other.joined.Format("2006-01-02 15:04:05")))
	c.msg(fmt.Sprintf("since: %s", time.Since(other.joined)))

	if other.room != nil {
		c.msg(fmt.Sprintf("room: %s", other.room.name))
	}
}

func (c *client) join(r *room) {
	c.leave()
	c.room = r
	r.addMember(c)
}

func (c *client) leave() {
	if c.room == nil {
		return
	}
	oldRoom := c.room
	c.room = nil
	oldRoom.removeMember(c)
}

func (c *client) Equal(other *client) bool {
	return c.conn.RemoteAddr() == other.conn.RemoteAddr()
}

func (c *client) quit() {
	c.commands <- command{
		id:     CMD_QUIT,
		client: c,
	}
}
