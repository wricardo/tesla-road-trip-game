package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

func main() {
	url := flag.String("url", "ws://localhost:9191/ws?session=9c1a", "WebSocket URL")
	flag.Parse()

	c, _, err := websocket.DefaultDialer.Dial(*url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, payload, err := c.ReadMessage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				return
			}

			if !json.Valid(payload) {
				continue
			}

			var compact bytes.Buffer
			if err := json.Compact(&compact, payload); err != nil {
				continue
			}
			fmt.Println(compact.String())
		}
	}()

	select {
	case <-sigCh:
	case <-done:
	}
}
