package udp

import (
    "testing"
)

func BenchmarkSend(b *testing.B) {
    // setup
    sendConfig := SendConfig{
        DestAddr:   "127.0.0.1",
        DestPort:   PORT,
    }
    send, err := NewSend(sendConfig)
    if err != nil {
        b.Fatal(err)
    }

    b.ResetTimer()

    // run
    send.run(0, 0, 100000) // 100k packets at unbound rate
}
