package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

// status codes of operations
const (
	SuccessCode = 200
	ErrorCode   = 500
	TimeoutCode = 600
)

type operationStatus struct {
	code    int
	details string
}

type request struct {
	ActionName string `json:"actionName"`
	Param      string `json:"param"`
}

type response struct {
	Ok   bool   `json:"ok"`
	Body string `json:"body"`
}

type client struct {
	name, addr string
	conn       *net.TCPConn
	reader     *bufio.Reader
	encoder    *json.Encoder
	lobby      *lobby
	opStatus   chan *operationStatus
	*commChannels
}

func newClient(conn *net.TCPConn, channels *commChannels) *client {
	return &client{
		addr:         conn.LocalAddr().String(),
		conn:         conn,
		reader:       bufio.NewReader(conn),
		encoder:      json.NewEncoder(conn),
		opStatus:     make(chan *operationStatus),
		commChannels: channels,
	}
}

func (c *client) handleConnection() {
	defer func() {
		if action, ok := requestHandler["leavelobby"]; ok {
			action.client(c, "")
		} else {
			c.err <- fmt.Errorf("on missing the 'leavelobby' action from the handler map")
		}
		c.connClose <- c.conn
	}()
	c.conn.SetKeepAlive(true)
	c.conn.SetKeepAlivePeriod(3 * time.Second)
	for {
		req, err := c.read()
		if err != nil {
			if err != io.EOF {
				c.err <- fmt.Errorf("on parsing a command from '%s': %v", c.addr, err)
				continue
			}
			break
		}
		if handlerFunc, ok := requestHandler[req.ActionName]; ok {
			err = handlerFunc.client(c, req.Param)
		} else {
			err = fmt.Errorf("on unknown command from %s", c.addr)
		}
		sendingErr := c.sendResponseToUser(err)
		if err != nil {
			c.err <- err
		}
		if sendingErr != nil {
			c.err <- sendingErr
		}
	}
}

func (c *client) read() (*request, error) {
	data, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	req := &request{}
	if err := json.Unmarshal(data[:len(data)], req); err != nil {
		return nil, err
	}
	return req, nil
}

func (c *client) write(resp *response) error {
	if err := c.encoder.Encode(resp); err != nil {
		return fmt.Errorf("on encoding a response: %v", err)
	}
	return nil
}

func (c *client) sendOperationStatus(status int, details string) {
	go func() {
		select {
		case c.opStatus <- &operationStatus{status, details}:
		case <-time.After(2 * time.Second):
		}
	}()
}

func (c *client) sendRequestToChat(actionName, param string) {
	c.chatIncoming <- &clientIncoming{actionName, param, c}
}

func (c *client) sendResponseToUser(err error) error {
	var opStatus *operationStatus
	if err != nil {
		// to do: do not show every error datail
		opStatus = &operationStatus{code: ErrorCode, details: err.Error()}
	} else {
		select {
		case opStatus = <-c.opStatus:
		case <-time.After(2 * time.Second):
			opStatus = &operationStatus{code: TimeoutCode, details: "operation timeout hit"}
		}
	}
	resp := &response{Ok: true, Body: opStatus.details}
	switch {
	case opStatus.code >= TimeoutCode || opStatus.code >= ErrorCode:
		resp.Ok = false
	}
	return c.write(resp)
}
