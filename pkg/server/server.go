package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// TCPServer instance
type TCPServer struct {
	addr string
	ctx  context.Context
	log  *logrus.Logger
}

// NewTCPServer is a constructor for TCPServer
func NewTCPServer(addr string, log *logrus.Logger) (*TCPServer, context.CancelFunc) {
	ctx, closeServer := context.WithCancel(context.Background())
	return &TCPServer{
		addr: addr,
		ctx:  ctx,
		log:  log,
	}, closeServer
}

// Start listening for new connections
func (s *TCPServer) Start() {
	// setting up the listener
	listener, err := s.createListener()
	if err != nil {
		s.log.Fatal(err)
	}
	s.log.Infof("the server started listening on %s (tcp)", s.addr)
	chat := NewChat(s.log)
	for {
		select {
		case <-s.ctx.Done():
			if err := listener.Close(); err != nil {
				s.log.Fatalf("on attempting to shutdown the listener: %v", err)
			}
			return
		default:
			// setting up a deadline to not miss a message from the context
			if err := listener.SetDeadline(time.Now().Add(time.Second)); err != nil {
				s.log.Panicf("on setting a timeout for the listener: %v", err)
			}
			conn, err := listener.AcceptTCP()
			if err != nil {
				if os.IsTimeout(err) {
					continue
				}
				s.log.Errorf("on a new connection: %v", err)
				chat.CloseConnection(conn)
			}
			chat.AddClient(conn)
		}
	}
}

func (s *TCPServer) createListener() (*net.TCPListener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", s.addr)
	if err != nil {
		return nil, fmt.Errorf("on resolving the address: %v", err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("on creating the listener: %v", err)
	}
	return listener, nil
}

// CreateLogger creates logger instance
func CreateLogger(filename string, stdout bool) (*logrus.Logger, error) {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	logger := logrus.New()
	// MultiWriter creates a writer that duplicates its writes to all the provided writers
	var writer io.Writer
	if stdout {
		writer = io.MultiWriter(os.Stdout, logFile)
	} else {
		writer = logFile
	}
	logger.SetOutput(writer)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return logger, nil
}
