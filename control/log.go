package control

// logging system with websocket streaming

import (
    "bufio"
    "fmt"
    "net/http"
    "io"
    "log"
    "os"
    "strings"
    "github.com/gorilla/websocket"
)

var wsUpgrader websocket.Upgrader = websocket.Upgrader{}

type LogMsg struct {
    Line        string   `json:"line"`
}

type Logs struct {
    history     []LogMsg
    followers   map[logFollower]bool
}

type logWriter struct {
    logs        *Logs
}

type logFollower struct {
    logs        *Logs
    wsConn      *websocket.Conn
    writeChan   chan LogMsg
}

func NewLogs() (*Logs, error) {
    self := &Logs{
        followers: make(map[logFollower]bool),
    }

    return self, nil
}

func (self *Logs) Logger(prefix string) *log.Logger {
    logWriter := logWriter{logs: self}

    writer := io.MultiWriter(logWriter, os.Stderr)

    return log.New(writer, prefix, 0)
}

func (self logWriter) Write(buf[]byte) (int, error) {
    line := string(buf)

    logMsg := LogMsg{
        Line: strings.TrimRight(line, "\n"),
    }

    self.logs.write(logMsg)

    return len(buf), nil
}

func (self *Logs) readr(reader *bufio.Reader) {
    for {
        if line, err := reader.ReadString('\n'); err != nil {
            log.Printf("Logs.reader: %v\n", err)
            return
        } else {
            self.write(LogMsg{line})
        }
    }
}

func (self *Logs) subscribe(follower logFollower) {
    log.Printf("Logs.follow: %v\n", follower)

    self.followers[follower] = true // XXX: race

    // sync up
    for _, msg := range self.history { // XXX: race
        follower.write(msg)
    }
}

func (self *Logs) unsubscribe(follower logFollower) {
    // XXX: race?
    delete(self.followers, follower)
}

// Called directly by the logger to distribute logs lines out to followers
func (self *Logs) write(msg LogMsg) {
    self.history = append(self.history, msg) // XXX: race

    for follower, _ := range self.followers { // XXX: race
        follower.write(msg)
    }
}

// http.Handler interface
func (self *Logs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    wsConn, err := wsUpgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("Logs.ServeHTTP: websocketUpgrader.Upgrade: %v\n", err)

        w.WriteHeader(400)
        return
    }

    follower := logFollower{
        logs:       self,
        wsConn:     wsConn,
        writeChan:  make(chan LogMsg, 100), // buffered
    }

    go follower.reader()
    go follower.writer()

    self.subscribe(follower)
}

func (self logFollower) error(err error) {
    log.Printf("logFollower %v: %v\n", self, err)

    self.logs.unsubscribe(self)
}

func (self logFollower) write(msg LogMsg) {
    select {
    case self.writeChan <- msg:

    default:
        log.Printf("logFollower.send %v: dropped write: %v\n", self)
    }
}

// service read messages, limited to websocket-internal ping handling
func (self logFollower) reader() {
    for {
        if _, _, err := self.wsConn.ReadMessage(); err != nil {
            self.error(fmt.Errorf("wsConn.ReadMessage: %v", err))
            break
        } else {
            // ignore
        }
    }
}

// ship distributed messages to clients
func (self logFollower) writer() {
    for msg := range self.writeChan {
        if err := self.wsConn.WriteJSON(msg); err != nil {
            self.error(fmt.Errorf("wsConn.WriteMessage: %v", err))
            break
        }
    }
}
