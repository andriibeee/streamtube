package queue

import (
	"log"
	"net"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type Queue struct {
	hub       *ConnectionsHub
	playlists *PlaylistHub
	es        *EventStream
}

func NewQueue(hub *ConnectionsHub, playlists *PlaylistHub, es *EventStream) Queue {
	return Queue{
		hub:       hub,
		playlists: playlists,
		es:        es,
	}
}

func (q *Queue) ProcessConnection(username string, conn net.Conn) {
	defer q.hub.OnClose(username)
	q.hub.OnConnection(username, conn)
	for {
		_, op, err := wsutil.ReadClientData(conn)
		if err != nil {
			log.Println(err)
			break
		}
		if op == ws.OpClose {
			break
		}
	}
}
