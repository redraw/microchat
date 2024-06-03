package cmd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

type Client struct {
	nick       string
	useTLS     bool
	skipVerify bool
	autojoin   string
	conn       net.Conn
}

const (
	PROMPT = "> "
)

func newConn(addr string, useTLS bool, skipVerify bool) (net.Conn, error) {
	if useTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: skipVerify,
		}
		return tls.Dial("tcp", addr, tlsConfig)
	}

	return net.Dial("tcp", addr)
}

func NewClient(addr string, useTLS, skipVerify bool, nick string, autojoin string) (*Client, error) {
	conn, err := newConn(addr, useTLS, skipVerify)

	if err != nil {
		return nil, err
	}

	client := &Client{
		useTLS:     useTLS,
		skipVerify: skipVerify,
		nick:       nick,
		autojoin:   autojoin,
		conn:       conn,
	}

	return client, nil
}

func (c *Client) Run() {
	defer c.conn.Close()
	fmt.Println("Connected to server.")

	go c.readMessages()

	if c.nick != "" {
		_, err := c.conn.Write([]byte("/nick " + c.nick + "\n"))
		if err != nil {
			log.Printf("error sending nickname to server: %v", err)
		}
	}

	if c.autojoin != "" {
		_, err := c.conn.Write([]byte("/join " + c.autojoin + "\n"))
		if err != nil {
			log.Printf("error sending autojoin to server: %v", err)
		}
	}

	c.handleUserInput()
}

func (c *Client) readMessages() {
	reader := bufio.NewReader(c.conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error reading from server: %v", err)
		}
		if err == io.EOF {
			fmt.Println("Connection closed by server.")
			return
		}
		fmt.Print("\r" + msg + PROMPT)
	}
}

func (c *Client) handleUserInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(PROMPT)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error reading from stdin: %v\n", err)
			continue
		}
		input = strings.TrimSpace(input)

		if input == "/quit" {
			c.conn.Close()
			fmt.Println("Connection closed. Goodbye!")
			return
		}

		_, err = c.conn.Write([]byte(input + "\n"))
		if err != nil {
			fmt.Printf("error sending message to server: %v\n", err)
		}
	}
}
