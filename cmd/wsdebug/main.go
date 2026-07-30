package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type wsMessage struct {
	SessionID string          `json:"session_id"`
	Event     string          `json:"event"`
	GameState json.RawMessage `json:"game_state"`
	Data      json.RawMessage `json:"data"`
}

func main() {
	url := flag.String("url", "ws://localhost:9191/ws?session=9c1a", "WebSocket URL")
	flag.Parse()

	c, _, err := websocket.DefaultDialer.Dial(*url, nil)
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer c.Close()

	log.Printf("connected: %s", *url)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			msgType, payload, err := c.ReadMessage()
			if err != nil {
				log.Printf("read error: %v", err)
				return
			}

			ts := time.Now().Format(time.RFC3339Nano)
			fmt.Printf("[%s] type=%d bytes=%d\n", ts, msgType, len(payload))

			var m wsMessage
			if err := json.Unmarshal(payload, &m); err == nil {
				fmt.Printf("  session=%s event=%s\n", m.SessionID, m.Event)
			}
			fmt.Printf("  raw=%s\n", string(payload))
		}
	}()

	select {
	case <-sigCh:
		log.Printf("received interrupt, shutting down")
	case <-done:
	}
}

