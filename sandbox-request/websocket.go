// websocket pipe
package main

import (
	"crypto/tls"

	"github.com/gofiber/websocket/v2"
	gorilla "github.com/gorilla/websocket"
)

func Proxy(c *websocket.Conn, endpoint string) error {
	// connect to upstream

	// headers := http.Header{}
	// headers.Add("Authorization", "Basic a2FzbV91c2VyOmhlYWRsZXNz")
	// headers.Add("Sec-WebSocket-Origin", "https://localhost")
	// headers.Add("Sec-WebSocket-Protocol", "binary")

	dialer := gorilla.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := dialer.Dial(endpoint, nil)
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
