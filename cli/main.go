package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kotsmile/go-dessage/server"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

func main() {
	user := flag.String("user", "", "name of user")
	addr := flag.String("addr", "", "address of the server")

	flag.Parse()

	s := server.NewServer(*addr, *user)
	s.WithOnMessage(func(message server.Message) {
		switch message.Type {
		case server.ConnectType:
			fmt.Printf("%s connected\n", message.User)
		case server.DisconnectType:
			fmt.Printf("%s disconnected\n", message.User)
		case server.SendType:
			fmt.Printf("%s> %s\n", message.User, message.Text)
		}
	})

	if err := s.ListenAndAccept(); err != nil {
		panic(err)
	}

	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		text = text[:len(text)-1]

		connectCmd := "connect "
		exitCmd := "exit"

		if strings.HasPrefix(text, connectCmd) {
			addr := text[len(connectCmd):]
			if err := s.Dial(addr); err != nil {
				fmt.Printf("ERROR: failed to connect to %s: %v\n", addr, err)
				continue
			}
			time.Sleep(1 * time.Second)

		} else if strings.HasPrefix(text, exitCmd) {
			if err := s.Close(); err != nil {
				fmt.Printf("ERROR: failed to exit from the server: %v\n", err)
				continue
			}
			return
		} else {
			s.SendMessage(text, server.SendType)
		}

	}
}
