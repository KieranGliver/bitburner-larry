package main

import (
	"fmt"
	"os"

	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
	"time"

	tea "charm.land/bubbletea/v2"
)

func startTui() {
	store := &Store{}

	if err := store.Init(); err != nil {
		fmt.Printf("Can't init store: %v", err)
	}

	m := newModel(store)

	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Unable to run tui: %v", err)
		os.Exit(1)
	}
}

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

// run starts a http.Server for the passed in address
// with all requests handled by echoServer.
func run() error {
	if len(os.Args) < 2 {
		return errors.New("please provide an address to listen on as the first argument")
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}
	log.Printf("listening on ws://%v", l.Addr())

	s := &http.Server{
		Handler: echoServer{
			logf: log.Printf,
		},
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
