package api

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsPingInterval/wsPongWait/wsWriteWait implement the standard gorilla
// heartbeat pattern: ping the peer well inside how long we're willing to
// wait for a pong, so a connection that's actually dead (peer crashed, or
// an intermediary silently dropped the TCP connection) is detected and
// closed in seconds rather than only surfacing as an EOF whenever the OS
// or a proxy eventually reaps it. Without this, a push-only, otherwise-idle
// socket can sit "connected" long after the peer is gone.
const (
	wsPingInterval = 30 * time.Second
	wsPongWait     = 60 * time.Second
)

// wsWriteWait bounds every write, pings and messages alike. A peer that has
// stopped reading -- a tab asleep behind Cloudflare, a till whose network
// died -- would otherwise let a write block forever once the TCP buffers
// fill. A var only so tests can shorten it.
var wsWriteWait = 10 * time.Second

// wsConn wraps a gorilla *websocket.Conn with a write mutex.
//
// gorilla/websocket allows one concurrent reader and one concurrent writer,
// but not two concurrent writers -- a second write while another is in
// flight panics with "concurrent write to websocket connection". Both hubs
// fan the same connection out to many goroutines (every handler that calls
// broadcastAll/notify), so every write has to go through this lock. Reads
// stay on the bare Conn in the handler's read loop, which is a separate
// goroutine and allowed to run alongside a write.
type wsConn struct {
	conn *websocket.Conn
	wmu  sync.Mutex
}

func newWSConn(conn *websocket.Conn) *wsConn {
	return &wsConn{conn: conn}
}

// writeJSON sends one message within wsWriteWait. A socket that can't take
// it is closed: a write that timed out leaves it unusable anyway, and
// closing ends the handler's read loop, which takes the peer out of its hub
// so later pushes don't wait on it again.
func (c *wsConn) writeJSON(v any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
	err := c.conn.WriteJSON(v)
	if err != nil {
		_ = c.conn.Close()
	}
	return err
}

func (c *wsConn) close() error {
	return c.conn.Close()
}

// writePing sends a control-frame ping, going through the same write mutex
// as writeJSON so it can't race a concurrent WriteJSON/WriteMessage call.
func (c *wsConn) writePing() error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsWriteWait))
}

// startHeartbeat arms the read-deadline/pong-handler pair and starts a
// goroutine writing periodic pings, so a dead peer is detected within
// roughly wsPongWait instead of waiting on the OS/an intermediary to
// eventually notice. Callers must call conn.SetReadDeadline once up front
// (done here) before their read loop starts, and stop the returned
// goroutine by closing done once the read loop exits.
func (c *wsConn) startHeartbeat(done <-chan struct{}) {
	c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(wsPongWait))
		return nil
	})

	go func() {
		ticker := time.NewTicker(wsPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := c.writePing(); err != nil {
					return
				}
			}
		}
	}()
}
