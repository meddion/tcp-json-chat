// +build integration

package server

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"
	"time"
)

// to run it type: go test -tags=integration ./...
func TestTCPServerIntegration(t *testing.T) {
	// creates a logger
	log, err := CreateLogger("../../testdata/server_test.log", false)
	if err != nil {
		t.Fatal("on creating a logger: ", err)
		os.Exit(0)
	}
	addr := ":6969"
	// starting the server for testing purposes
	s, close := NewTCPServer(addr, log)
	go s.Start()
	defer close()
	time.Sleep(time.Second) // give the server a second to start working

	testTable := []struct {
		test          string
		input, output []byte
	}{
		{
			`Sending "login" request`,
			[]byte(`{"actionName":"login","param":"medion"}`),
			[]byte(`{"ok":true,"body":""}`),
		},
		{
			`Sending "addlobby" request`,
			[]byte(`{"actionName":"addlobby","param":"testLobby"}`),
			[]byte(`{"ok":true,"body":""}`),
		},
		{
			`Sending "joinlobby" request`,
			[]byte(`{"actionName":"joinlobby","param":"testLobby"}`),
			[]byte(`{"ok":true,"body":"medion has joined to the chat."}`),
		},
		{
			`Sending "broadcast" request`,
			[]byte(`{"actionName":"broadcast","param":"Hello, World!"}`),
			[]byte(`{"ok":true,"body":"medion: Hello, World!"}`),
		},
		{
			`Sending "leavelobby" request`,
			[]byte(`{"actionName":"leavelobby","param":""}`),
			[]byte(`{"ok":true,"body":""}`),
		},
	}
	// connecting to the server
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal("on connecting to TCP server:", err)
	}
	defer conn.Close()
	readWriteTimeout := 3 * time.Second
	for _, testCase := range testTable {
		t.Run(testCase.test, func(t *testing.T) {
			if err := conn.SetWriteDeadline(time.Now().Add(readWriteTimeout)); err != nil {
				t.Fatal("on setting a deadline on writing from a connection:", err)
			}
			if _, err := conn.Write(append(testCase.input, '\n')); err != nil {
				t.Fatal("on writing to TCP server:", err)
			}
			out := make([]byte, 1024)
			if err := conn.SetReadDeadline(time.Now().Add(readWriteTimeout)); err != nil {
				t.Fatal("on setting a deadline on reading from a connection:", err)
			}
			if _, err := conn.Read(out); err != nil {
				t.Fatal("on reading from a connection:", err)
			}
			testCase.output = append(testCase.output, '\n')
			if bytes.Compare(out[:len(testCase.output)], testCase.output) != 0 {
				t.Fatal(fmt.Sprintf(`The expected value: "%s" - the value we got: "%s"`,
					string(testCase.output), string(out)))
			}
		})
	}
}
