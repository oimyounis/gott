package gott

import (
	"log"
	"net"
)

type Broker struct {
	address  string
	listener net.Listener
	clients  []*Client
}

func NewBroker(address string) *Broker {
	return &Broker{address: address, listener: nil}
}

func (b *Broker) Listen() error {
	l, err := net.Listen("tcp", b.address)
	if err != nil {
		return err
	}
	defer l.Close()

	b.listener = l
	log.Println("Broker listening on " + b.listener.Addr().String())

	for {
		conn, err := b.listener.Accept()
		if err != nil {
			log.Printf("Couldn't accept connection: %v\n", err)
		} else {
			go b.handleConnection(conn)
		}
	}
	return nil
}

func (b *Broker) addClient(conn net.Conn) *Client {
	c := &Client{connection: conn, connected: true}
	b.clients = append(b.clients, c)
	return c
}

func (b *Broker) handleConnection(conn net.Conn) {
	log.Printf("Accepted connection from %v\n", conn.RemoteAddr().String())
	client := b.addClient(conn)
	go client.listen()
}
