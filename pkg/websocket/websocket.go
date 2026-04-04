// Package websocket provides WebSocket support using Go's stdlib HTTP hijacking.
// No external dependencies — implements the WebSocket handshake and framing protocol.
package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// Frame opcodes
	OpText   = 1
	OpBinary = 2
	OpClose  = 8
	OpPing   = 9
	OpPong   = 10

	wsGUID = "258EAFA5-E914-47DA-95CA-5AB5A11735B4"
)

// Conn represents a WebSocket connection.
type Conn struct {
	conn   net.Conn
	rw     *bufio.ReadWriter
	mu     sync.Mutex
	closed bool
}

// Handler is a function that handles a WebSocket connection.
type Handler func(conn *Conn)

// Upgrade upgrades an HTTP connection to a WebSocket connection.
func Upgrade(w http.ResponseWriter, r *http.Request, handler Handler) error {
	if !isWebSocketUpgrade(r) {
		http.Error(w, "Not a WebSocket request", http.StatusBadRequest)
		return errors.New("websocket: not a websocket upgrade request")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Missing Sec-WebSocket-Key", http.StatusBadRequest)
		return errors.New("websocket: missing Sec-WebSocket-Key")
	}

	acceptKey := computeAcceptKey(key)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return errors.New("websocket: response does not support hijacking")
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return fmt.Errorf("websocket: hijack failed: %w", err)
	}

	// Send upgrade response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	if _, err := rw.WriteString(response); err != nil {
		conn.Close()
		return err
	}
	if err := rw.Flush(); err != nil {
		conn.Close()
		return err
	}

	ws := &Conn{conn: conn, rw: rw}

	go func() {
		defer ws.Close()
		handler(ws)
	}()

	return nil
}

// ReadMessage reads the next message from the WebSocket.
func (c *Conn) ReadMessage() (opcode int, payload []byte, err error) {
	for {
		op, data, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}

		switch op {
		case OpClose:
			c.Close()
			return OpClose, nil, io.EOF
		case OpPing:
			c.writeFrame(OpPong, data)
			continue
		case OpPong:
			continue
		default:
			return op, data, nil
		}
	}
}

// ReadText reads the next text message.
func (c *Conn) ReadText() (string, error) {
	_, data, err := c.ReadMessage()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteText sends a text message.
func (c *Conn) WriteText(msg string) error {
	return c.writeFrame(OpText, []byte(msg))
}

// WriteBinary sends a binary message.
func (c *Conn) WriteBinary(data []byte) error {
	return c.writeFrame(OpBinary, data)
}

// WriteJSON sends a JSON string message.
func (c *Conn) WriteJSON(data string) error {
	return c.WriteText(data)
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.writeFrameUnsafe(OpClose, nil)
	return c.conn.Close()
}

// SetDeadline sets the read/write deadline.
func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// RemoteAddr returns the remote address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Conn) readFrame() (opcode int, payload []byte, err error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.rw, header); err != nil {
		return 0, nil, err
	}

	opcode = int(header[0] & 0x0F)
	masked := header[1]&0x80 != 0
	length := int64(header[1] & 0x7F)

	switch length {
	case 126:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(c.rw, buf); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(buf))
	case 127:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(c.rw, buf); err != nil {
			return 0, nil, err
		}
		length = int64(binary.BigEndian.Uint64(buf))
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.rw, mask); err != nil {
			return 0, nil, err
		}
	}

	payload = make([]byte, length)
	if _, err := io.ReadFull(c.rw, payload); err != nil {
		return 0, nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return opcode, payload, nil
}

func (c *Conn) writeFrame(opcode int, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeFrameUnsafe(opcode, payload)
}

func (c *Conn) writeFrameUnsafe(opcode int, payload []byte) error {
	frame := []byte{byte(0x80 | opcode)}
	length := len(payload)

	switch {
	case length <= 125:
		frame = append(frame, byte(length))
	case length <= 65535:
		frame = append(frame, 126)
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(length))
		frame = append(frame, buf...)
	default:
		frame = append(frame, 127)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		frame = append(frame, buf...)
	}

	frame = append(frame, payload...)

	if _, err := c.rw.Write(frame); err != nil {
		return err
	}
	return c.rw.Flush()
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// --- Hub for managing multiple connections ---

// Hub manages a set of WebSocket connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Conn]bool
	rooms   map[string]map[*Conn]bool
}

// NewHub creates a WebSocket connection hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Conn]bool),
		rooms:   make(map[string]map[*Conn]bool),
	}
}

// Add registers a connection.
func (h *Hub) Add(conn *Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
}

// Remove unregisters a connection.
func (h *Hub) Remove(conn *Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	for _, room := range h.rooms {
		delete(room, conn)
	}
	h.mu.Unlock()
}

// Join adds a connection to a room.
func (h *Hub) Join(conn *Conn, room string) {
	h.mu.Lock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Conn]bool)
	}
	h.rooms[room][conn] = true
	h.mu.Unlock()
}

// Leave removes a connection from a room.
func (h *Hub) Leave(conn *Conn, room string) {
	h.mu.Lock()
	if h.rooms[room] != nil {
		delete(h.rooms[room], conn)
	}
	h.mu.Unlock()
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients {
		conn.WriteText(msg)
	}
}

// BroadcastTo sends a message to all clients in a room.
func (h *Hub) BroadcastTo(room, msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[room]; ok {
		for conn := range clients {
			conn.WriteText(msg)
		}
	}
}

// Count returns the total number of connected clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	n := len(h.clients)
	h.mu.RUnlock()
	return n
}

// RoomCount returns the number of clients in a room.
func (h *Hub) RoomCount(room string) int {
	h.mu.RLock()
	n := len(h.rooms[room])
	h.mu.RUnlock()
	return n
}
