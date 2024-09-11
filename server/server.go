package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Peer struct {
	remoteAddr string
	conn       net.Conn
	encoder    *json.Encoder
	outbound   bool
}

func NewPeer(conn net.Conn, outbound bool) Peer {
	return Peer{
		conn:       conn,
		remoteAddr: conn.RemoteAddr().String(),
		outbound:   outbound,
		encoder:    json.NewEncoder(conn),
	}
}

func (p Peer) Addr() string {
	return p.remoteAddr
}

type Server struct {
	addr     string
	username string

	listener  net.Listener
	messageCh chan Message
	peers     sync.Map // map[string]Peer
	log       *log.Entry

	onMessage func(Message)
}

func NewServer(addr string, username string) *Server {
	return &Server{
		addr:      addr,
		username:  username,
		messageCh: make(chan Message, 1024),

		log: log.WithField("server", addr),
	}
}

func (s *Server) WithOnMessage(onMessage func(Message)) {
	s.onMessage = onMessage
}

func (s *Server) Addr() string {
	return s.addr
}

func (s *Server) ListenAndAccept() error {
	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen addr %s: %v", s.addr, err)
	}

	go s.listenAndAccept()
	go s.handleMessage()

	return nil
}

func (s *Server) Dial(addr string) error {
	s.log.WithField("addr", addr).Debugln("dialing")

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", addr, err)
	}

	go s.handleConn(conn, false)

	return nil
}

func (s *Server) SendMessage(text string, messageType MessageType) {
	s.messageCh <- Message{
		User:      s.username,
		Text:      text,
		Timestamp: time.Now().Unix(),
		Type:      messageType,
		internal:  true,
	}
}

func (s *Server) Close() error {
	if s.listener == nil {
		return nil
	}

	s.SendMessage("disconnect", DisconnectType)

	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %v", err)
	}

	return nil
}

func (s *Server) listenAndAccept() {
	s.log.WithField("address", s.addr).Debugln("start server")

	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			s.log.Errorf("failed to accept connection: %v", err)
			continue
		}

		go s.handleConn(conn, true)
	}
}

func (s *Server) handleConn(conn net.Conn, outbound bool) {
	remoteAddr := conn.RemoteAddr().String()
	s.log.WithField("remoteAddr", remoteAddr).Debugln("new connection")

	defer func() {
		s.log.WithField("remoteAddr", remoteAddr).Debugln("close connection")
		s.peers.Delete(remoteAddr)
		if err := conn.Close(); err != nil {
			s.log.WithField("remoteAddr", remoteAddr).Errorf("failed to close connection: %v", err)
			return
		}
	}()

	peer := NewPeer(conn, outbound)
	s.peers.Store(remoteAddr, &peer)

	if !outbound {
		s.SendMessage("connect", ConnectType)
	}

	decoder := json.NewDecoder(conn)
	for {
		message := Message{}
		err := decoder.Decode(&message)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			s.log.WithField("remoteAddr", remoteAddr).Errorf("failed to receive message: %v", err)
			continue
		}

		message.Address = remoteAddr
		s.messageCh <- message
	}
}

func (s *Server) handleMessage() {
	for {
		message := <-s.messageCh
		if !message.internal && s.onMessage != nil {
			s.onMessage(message)
		}

		s.peers.Range(func(remoteAddr_ any, peer_ any) bool {
			remoteAddr, _ := remoteAddr_.(string)
			peer, _ := peer_.(*Peer)

			if remoteAddr == message.Address {
				return true
			}

			if err := peer.encoder.Encode(message); err != nil {
				s.log.WithField("remoteAddr", remoteAddr).
					WithField("message", message).
					Errorf("failed to send message: %v", err)
				return true
			}

			return true
		})
	}
}
