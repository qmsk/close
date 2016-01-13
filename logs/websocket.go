package logs

import (
    "fmt"
    "net/http"
    "log"
    "github.com/gorilla/websocket"
)

var wsUpgrader websocket.Upgrader = websocket.Upgrader{}

// http.Handler interface
func (self *Logs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    wsConn, err := wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Logs.ServeHTTP: websocketUpgrader.Upgrade: %v\n", err)

        w.WriteHeader(400)
        return
    }

    follower := newFollower(self)

    go websocketReader(follower, wsConn)
    go websocketWriter(follower, wsConn)
}

// service read messages, limited to websocket-internal ping handling
// XXX: assume that ReadMessage() will error out once wsConn.Close()?
func websocketReader(follower logFollower, wsConn *websocket.Conn) {
    for {
        if _, _, err := wsConn.ReadMessage(); err != nil {
            follower.close(fmt.Errorf("wsConn.ReadMessage: %v", err))
            break
        } else {
            // ignore
        }
    }
}

// ship distributed messages to clients
// will also close the websocket once the writeChan is closed by logFollower.run()
func websocketWriter(follower logFollower, wsConn *websocket.Conn) {
    defer wsConn.Close()

    for msg := range follower.writeChan {
        if err := wsConn.WriteJSON(msg); err != nil {
            follower.close(fmt.Errorf("wsConn.WriteMessage: %v", err))
            break
        }
    }
}
