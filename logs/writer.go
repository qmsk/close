package logs

import (
    "strings"
)

type logWriter struct {
    writeChan   chan LogMsg
}

// Act as a Logger's io.Writer, processing logged messages
// Assumes that each log.Logger.Printf() etc call results in a single Write()
// This must be goroutine-safe and avoid blocking, since that would block any callers of Logger.Printf()!
func (self logWriter) Write(buf[]byte) (int, error) {
    line := string(buf)

    logMsg := LogMsg{
        Line: strings.TrimRight(line, "\n"),
    }

    self.writeChan <- logMsg

    return len(buf), nil
}
