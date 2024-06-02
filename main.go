package main

import (
	"flag"
	"log"
)

func main() {
	port := flag.Int("port", 8888, "port to run the server on")
	flag.Parse()

	server := newServer()
	if err := server.listen(*port); err != nil {
		log.Fatalf("failed to start server: %s", err.Error())
	}
}
