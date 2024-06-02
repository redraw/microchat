package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

type client struct {
	conn     net.Conn
	nick     string
	room     *room
	commands chan<- command
}

func (c *client) readInput() {
	defer c.quit()

	for {
		msg, err := bufio.NewReader(c.conn).ReadString('\n')
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
					args:   args,
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
	c.conn.Write([]byte("err: " + err.Error() + "\n"))
}

func (c *client) msg(msg string) {
	c.conn.Write([]byte("> " + msg + "\n"))
}

func (c *client) quit() {
	c.commands <- command{
		id:     CMD_QUIT,
		client: c,
	}
}

func (c *client) Equal(other *client) bool {
	return c.conn.RemoteAddr() == other.conn.RemoteAddr()
}
