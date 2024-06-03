package cmd

import (
	"flag"
	"log"

	"github.com/redraw/microchat/lib"
)

func Run() {
	mode := flag.String("mode", "client", "mode to run the app (server or client)")
	addr := flag.String("addr", ":8888", "port to run the server on")
	useTLS := flag.Bool("tls", false, "enable TLS")
	skipVerify := flag.Bool("skip-verify", false, "skip TLS certificate verification on client")
	certFile := flag.String("cert", "server.crt", "TLS certificate file")
	keyFile := flag.String("key", "server.key", "TLS key file")
	nick := flag.String("nick", "", "nickname to use in chat")
	autojoin := flag.String("autojoin", "", "channel to join")
	flag.Parse()

	switch *mode {
	case "server":
		server := lib.NewServer(*addr, *useTLS, *certFile, *keyFile)
		if err := server.Run(); err != nil {
			log.Fatalf("failed to start server: %s", err.Error())
		}
	case "client":
		client, err := NewClient(*addr, *useTLS, *skipVerify, *nick, *autojoin)
		if err != nil {
			log.Fatalf("failed to create client: %s", err.Error())
		}
		client.Run()
	default:
		log.Fatalf("invalid mode: %s", *mode)
	}
}
