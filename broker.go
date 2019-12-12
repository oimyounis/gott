package gott

import (
	"log"
	"net"
	"sync"
)

var (
	// 4 is MQTT v3.1.1
	SUPPORTED_PROTOCOL_VERSIONS = []byte{4}
)

var GOTT *Broker

type Broker struct {
	address  string
	listener net.Listener
	clients  map[string]*Client
	mutex    sync.RWMutex
}

func NewBroker(address string) *Broker {
	GOTT = &Broker{address: address, listener: nil, clients: map[string]*Client{}}
	return GOTT
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
	//return nil
}

func (b *Broker) addClient(client *Client) {
	b.mutex.RLock()
	if c, ok := b.clients[client.ClientId]; ok {
		// disconnect existing client
		log.Println("disconnecting existing client with id:", c.ClientId)
		c.closeConnection()
	}
	b.mutex.RUnlock()

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.clients[client.ClientId] = client
}

func (b *Broker) removeClient(clientId string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	delete(b.clients, clientId)
}

func (b *Broker) handleConnection(conn net.Conn) {
	log.Printf("Accepted connection from %v", conn.RemoteAddr().String())
	//client := b.addClient(conn)
	c := &Client{connection: conn, connected: true}
	go c.listen()
}
