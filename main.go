package main

import (
	"os"
	"time"

	"github.com/kotsmile/go-dessage/server"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	server1 := server.NewServer(":3001", "user1")
	server2 := server.NewServer(":3002", "user2")
	server3 := server.NewServer(":3003", "user3")

	if err := server1.ListenAndAccept(); err != nil {
		log.Fatalf("failed to server server1: %v", err)
	}
	if err := server2.ListenAndAccept(); err != nil {
		log.Fatalf("failed to start server2: %v", err)
	}
	if err := server3.ListenAndAccept(); err != nil {
		log.Fatalf("failed to start server3: %v", err)
	}

	log.Info("waiting for starting servers")
	time.Sleep(5 * time.Second)

	if err := server2.Dial(server1.Addr()); err != nil {
		log.Fatalf("failed to dial server2 to server1: %v", err)
	}
	if err := server3.Dial(server1.Addr()); err != nil {
		log.Fatalf("failed to dial server3 to server1: %v", err)
	}

	log.Info("waiting for dialing")
	time.Sleep(5 * time.Second)

	server1.SendMessage("hello all", server.ConnectType)
	time.Sleep(1 * time.Second)
	server2.SendMessage("hello u", server.ConnectType)

	select {}
}
