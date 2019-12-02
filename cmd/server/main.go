package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/meddion/tcp-server/pkg/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// setting up the flags
	stdlog := flag.Bool("stdlog", true, "a logger param")
	logfile := flag.String("logfile", "./logs/server.log", "defines a log file")
	addr := flag.String("addr", ":3030", "listening address")
	flag.Parse()

	// creating a logger
	log, err := server.CreateLogger(*logfile, *stdlog)
	if err != nil {
		fmt.Printf("failed to create a logger: %v.\n", err)
		os.Exit(1)
	}

	// listening for new connections
	s, closeServer := server.NewTCPServer(*addr, log)
	go s.Start()

	// checking for OS shutdown signals to close the server
	listenForShutdownSignals(closeServer, log)
}

func listenForShutdownSignals(closeServer context.CancelFunc, log *logrus.Logger) {
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	log.Infof("caught `%+v` signal", sig)
	closeServer()
	log.Infof("the server's gently stopped its session")
}
