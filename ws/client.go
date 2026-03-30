package ws

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (we only expect pong frames).
	maxMessageSize = 512

	// Send buffer size per client.
	sendBufferSize = 256
)

// Client represents a single WebSocket connection from an aggregator.
type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	aggregatorID uint
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, aggregatorID uint) *Client {
	return &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, sendBufferSize),
		aggregatorID: aggregatorID,
	}
}

// ReadPump reads messages from the WebSocket connection.
// We don't expect real messages — just pings/pongs to keep alive.
// When the client disconnects, it unregisters from the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c.aggregatorID, c)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[WS Client] Read error for agg %d: %v", c.aggregatorID, err)
			}
			break
		}
	}
}

// WritePump sends messages from the hub to the WebSocket connection.
// Also handles periodic pings to keep the connection alive.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("[WS Client] Write error for agg %d: %v", c.aggregatorID, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
