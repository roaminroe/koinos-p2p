package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	koinosmq "github.com/koinos/koinos-mq-golang"
	"github.com/koinos/koinos-p2p/internal/node"
	"github.com/koinos/koinos-p2p/internal/rpc"
)

func main() {
	var addr = flag.String("listen", "/ip4/127.0.0.1/tcp/8888", "The multiaddress on which the node will listen")
	var seed = flag.Int("seed", 0, "Random seed with which the node will generate an ID")
	var peer = flag.String("peer", "", "Address of a peer to which to connect")
	var amqpFlag = flag.String("a", "amqp://guest:guest@localhost:5672/", "AMQP server URL")

	flag.Parse()

	mq := koinosmq.NewKoinosMQ(*amqpFlag)
	mq.Start()

	host, _ := node.NewKoinosP2PNode(context.Background(), *addr, rpc.NewKoinosRPC(), int64(*seed))
	log.Printf("Starting node at address: %s\n", host.GetPeerAddress())

	// Connect to a peer
	if *peer != "" {
		log.Println("Connecting to peer and sending broadcast")
		peer, err := host.ConnectToPeer(*peer)
		if err != nil {
			panic(err)
		}

		go host.Protocols.Broadcast.InitiateProtocol(context.Background(), peer.ID)
	}

	// Wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Println("Shutting down node...")
	// Shut the node down
	host.Close()
}
