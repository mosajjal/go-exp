// websocket pipe
package main

import (
	"github.com/gofiber/websocket/v2"
	gorilla "github.com/gorilla/websocket"
)

func Proxy(c *websocket.Conn, endpoint string) error {
	// connect to upstream
	conn, _, err := gorilla.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// pipe messages
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			c.WriteMessage(websocket.BinaryMessage, msg)
		}
	}()
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return err
		}
		conn.WriteMessage(gorilla.BinaryMessage, msg)
	}
}
