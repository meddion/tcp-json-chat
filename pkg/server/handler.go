package server

import (
	"fmt"
	"strings"
)

type (
	clientHandlerFunc func(*client, string) error
	chatHandlerFunc   func(*Chat, *client, string)
)

type requestHandlers struct {
	client clientHandlerFunc
	chat   chatHandlerFunc
}

var requestHandler = map[string]requestHandlers{
	"broadcast": {
		client: func(c *client, msg string) error {
			c.sendRequestToChat("broadcast", fmt.Sprintf("%s: %s", c.name, strings.TrimSpace(msg)))
			return nil
		},
		chat: func(chat *Chat, client *client, msg string) { chat.broadcast(client, msg) },
	},
	"addlobby": {
		client: func(c *client, lobbyname string) error {
			if len(lobbyname) < 3 || len(lobbyname) > 32 {
				return fmt.Errorf("on creating a lobby")
			}
			c.sendRequestToChat("addlobby", lobbyname)
			return nil
		},
		chat: func(chat *Chat, client *client, lobbyname string) {
			chat.addNewLobby(lobbyname)
			client.sendOperationStatus(SuccessCode, "")
		},
	},
	"joinlobby": {
		client: func(c *client, lobbyname string) error {
			c.sendRequestToChat("joinlobby", lobbyname)
			return nil
		},
		chat: func(chat *Chat, client *client, lobbyname string) {
			for lobby := range chat.lobbies {
				if lobby.name == lobbyname {
					if client.lobby != nil {
						chat.removeClientFromLobby(client)
					}
					lobby.clients[client] = struct{}{}
					client.lobby = lobby
					chat.broadcast(client, fmt.Sprintf("%s has joined to the chat.", client.name))
					return
				}
			}
			client.sendOperationStatus(ErrorCode, "there's no such lobby on the server")
		},
	},
	"leavelobby": {
		client: func(c *client, msg string) error {
			if c.lobby == nil {
				return fmt.Errorf("on leaving from an unknown lobby")
			}
			c.sendRequestToChat("leavelobby", "")
			return nil
		},
		chat: func(chat *Chat, client *client, msg string) {
			chat.removeClientFromLobby(client)
			client.lobby = nil
			client.sendOperationStatus(SuccessCode, "")
		},
	},
	"login": {
		client: func(c *client, username string) error {
			if len(username) < 3 || len(username) > 32 {
				return fmt.Errorf("on authorizing a user")
			}
			c.name = username
			go func() { c.opStatus <- &operationStatus{code: SuccessCode, details: ""} }()
			return nil
		},
	},
}
