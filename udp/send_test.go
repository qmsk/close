package udp

import (
    "testing"
)

func BenchmarkSendUDP(b *testing.B) {
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
    send.run(0, 0, 1000000) // 1m packets at unbound rate
}

func BenchmarkSendIP(b *testing.B) {
    // setup
    sendConfig := SendConfig{
        SourceNet:      "127.0.1.0/24",
        SourcePortBits: 16,
        DestAddr:       "127.0.0.1",
        DestPort:       PORT,
    }
    send, err := NewSend(sendConfig)
    if err != nil {
        b.Fatal(err)
    }

    b.ResetTimer()

    // run
    send.run(0, 0, 1000000) // 1m packets at unbound rate
}
