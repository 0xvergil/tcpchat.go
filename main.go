package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"tcpChat/chat"
)

const (
	defaultPort = "8989"
	usage       = "[USAGE]: ./TCPChat $port"
)

func main() {
	port, ok := parseArgs(os.Args[1:])
	if !ok {
		fmt.Println(usage)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("cannot listen on port %s: %v", port, err)
	}
	defer ln.Close()

	fmt.Printf("Listening on the port :%s\n", port)
	if err := chat.NewServer().Serve(ln); err != nil {
		log.Fatal(err)
	}
}

func parseArgs(args []string) (port string, ok bool) {
	switch len(args) {
	case 0:
		return defaultPort, true
	case 1:
		var p int
		if _, err := fmt.Sscanf(args[0], "%d", &p); err != nil ||
			fmt.Sprint(p) != args[0] || p < 1 || p > 65535 {
			return "", false
		}
		return args[0], true
	}
	return "", false
}