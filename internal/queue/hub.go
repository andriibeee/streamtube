package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var ErrNoConnectionFound = errors.New("no connection found")

type ConnectionsHub struct {
	connections map[string]io.Writer
	mu          *sync.RWMutex
}

func NewConnectionsHub() ConnectionsHub {
	return ConnectionsHub{
		connections: map[string]io.Writer{},
		mu:          &sync.RWMutex{},
	}
}

func (hub *ConnectionsHub) OnConnection(username string, conn io.Writer) {
	defer hub.mu.Unlock()
	hub.mu.Lock()
	hub.connections[username] = conn
}

func (hub *ConnectionsHub) SendMessage(username string, message interface{}) error {
	defer hub.mu.RUnlock()
	hub.mu.RLock()
	conn, ok := hub.connections[username]
	if !ok {
		return fmt.Errorf("cannot deliver message: %w, username: %s", ErrNoConnectionFound, username)
	}
	msg, err := json.Marshal(message)
	if err != nil {
		return err // TODO: wrap with more context
	}
	return wsutil.WriteServerMessage(conn, ws.OpText, msg)
}

func (hub *ConnectionsHub) OnClose(username string) {
	defer hub.mu.Unlock()
	hub.mu.Lock()
	delete(hub.connections, username)
}
