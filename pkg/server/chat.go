package server

import (
	"net"

	"github.com/sirupsen/logrus"
)

type lobby struct {
	name    string
	clients map[*client]struct{}
}

type clientIncoming struct {
	actionName, param string
	client            *client
}

type commChannels struct {
	chatIncoming chan *clientIncoming
	connClose    chan interface{}
	err          chan error
}

// Chat instance
type Chat struct {
	lobbies map[*lobby]struct{}
	clients map[*client]struct{}
	log     *logrus.Logger
	commChannels
}

// NewChat is a constructor for chat struct
func NewChat(log *logrus.Logger) *Chat {
	chat := &Chat{
		lobbies: make(map[*lobby]struct{}),
		clients: make(map[*client]struct{}),
		log:     log,
		commChannels: commChannels{
			chatIncoming: make(chan *clientIncoming),
			connClose:    make(chan interface{}),
			err:          make(chan error),
		},
	}
	chat.addNewLobby("main")
	chat.listenForConnEvents()
	return chat
}

// AddClient adds a client to the slice of clients and start listening on him
func (c *Chat) AddClient(conn *net.TCPConn) {
	client := newClient(conn, &c.commChannels)
	c.clients[client] = struct{}{}
	go client.handleConnection()
}

// CloseConnection sends a signal to the channel to close the connection with a user
func (c *Chat) CloseConnection(conn *net.TCPConn) {
	c.connClose <- conn
}

func (c *Chat) addNewLobby(name string) {
	lobby := &lobby{name: name, clients: make(map[*client]struct{})}
	c.lobbies[lobby] = struct{}{}
}

func (c *Chat) removeClientFromLobby(client *client) {
	if client.lobby != nil {
		delete(client.lobby.clients, client)
	}
}

func (c *Chat) listenForConnEvents() {
	go func() {
		for {
			select {
			case req := <-c.chatIncoming:
				if action, ok := requestHandler[req.actionName]; ok {
					action.chat(c, req.client, req.param)
					continue
				}
				c.log.Panicf("on handling an unknown action '%s' from a client", req.actionName)
			case conn := <-c.connClose:
				c.closeConnHandler(conn)
			case err := <-c.err:
				c.log.Error(err)
			}
		}
	}()
}

func (c *Chat) broadcast(client *client, msg string) {
	if client.lobby == nil {
		client.sendOperationStatus(ErrorCode, "on broadcasting to an undefined lobby")
		return
	}
	for aClient := range client.lobby.clients {
		aClient.sendOperationStatus(SuccessCode, msg)
	}
}

func (c *Chat) closeConnHandler(conn interface{}) {
	if client, ok := conn.(*client); ok {
		delete(c.clients, client)
		conn = client.conn
	}
	tConn, _ := conn.(*net.TCPConn)
	if err := tConn.Close(); err != nil {
		c.log.Errorf("on attempt to close a connection: %v", err)
	}
}
