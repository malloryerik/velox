package velox

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/gorilla/websocket"
)

//Conn represents a single websocket connection
//being synchronised. Its ID is the the connections remote
//address.
type Conn interface {
	ID() string
	Connected() bool
	Wait()
}

type transport interface {
	connect(w http.ResponseWriter, r *http.Request) error
	send(upd *update) error
}

type conn struct {
	transport
	connected bool
	id        string
	uptime    time.Time
	version   int64
	waiter    sync.WaitGroup
}

func (c *conn) ID() string {
	return c.id
}

func (c *conn) Connected() bool {
	return c.connected
}

//Wait will block until the connection is closed.
func (c *conn) Wait() {
	c.waiter.Wait()
}

//=========================

type wsTrans struct {
	conn *websocket.Conn
}

func (ws *wsTrans) connect(w http.ResponseWriter, r *http.Request) error {
	conn, err := defaultUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return fmt.Errorf("cannot upgrade connection: %s", err)
	}
	ws.conn = conn
	for {
		//msgType, msgBytes, err
		if _, _, err := conn.ReadMessage(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func (ws *wsTrans) send(upd *update) error {
	return ws.conn.WriteJSON(upd)
}

//=========================

type evtSrcTrans struct {
	s *eventsource.Server
}

func (es *evtSrcTrans) connect(w http.ResponseWriter, r *http.Request) error {
	es.s = eventsource.NewServer()
	es.s.Gzip = true
	es.s.Handler("events").ServeHTTP(w, r)
	return nil
}

func (es *evtSrcTrans) send(upd *update) error {
	es.s.Publish([]string{"events"}, upd)
	return nil
}