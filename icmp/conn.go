package icmp

import (
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv4"
    "golang.org/x/net/ipv6"
    "net"
    "fmt"
)

type Conn struct {
    TargetAddr  *net.UDPAddr
    IcmpConn    *icmp.PacketConn
    Proto       Proto
}

type Proto struct {
    listenStr   string
    resolveStr  string
    ianaProto   int
    messageType icmp.Type
}

// IANA ICMP v4 protocol is 1
// IANA ICMP v6 protocol is 58
var (
    protocols = map[string]Proto {
        "ipv4": Proto {
            resolveStr:  "ip4",
            listenStr:   "udp4",
            ianaProto:   1,
            messageType: ipv4.ICMPTypeEcho,
        },
        "ipv6": Proto {
            resolveStr: "ip6",
            listenStr:  "udp6",
            ianaProto:  58,
            messageType: ipv6.ICMPTypeEchoRequest,
        },
    }
)

func NewConn(config PingConfig) (*Conn, error) {
    c := &Conn {
    }

    if proto, err := protocols[config.Proto]; !err {
        return nil, fmt.Errorf("protocol not registered: %v", err)
    } else {
        c.Proto = proto
    }

    if targetAddr, err := c.resolveTarget(config.Target); err != nil {
        return nil, fmt.Errorf("resolveTarget: %v", err)
    } else {
        c.TargetAddr = targetAddr
    }

    if icmpConn, err := c.icmpListen(); err != nil {
        return nil, fmt.Errorf("icmpListen: %v", err)
    } else {
        c.IcmpConn = icmpConn
    }

    return c, nil
}

func (c *Conn) resolveTarget(target string) (*net.UDPAddr, error) {
    if target == "" {
        return nil, fmt.Errorf("No target given")
    }

    if ipAddr, err := net.ResolveIPAddr(c.Proto.resolveStr, target); err != nil {
        return nil, err
    } else {
        // unprivileged icmp mode uses SOCK_DGRAM
        udpAddr := &net.UDPAddr{IP: ipAddr.IP, Zone: ipAddr.Zone}

        return udpAddr, nil
    }
}

func (c *Conn) icmpListen() (*icmp.PacketConn, error) {
    return icmp.ListenPacket(c.Proto.listenStr, "")
}

func (c *Conn) Write(wm icmp.Message) (error) {
    wb, err := wm.Marshal(nil)
    if err != nil {
        return fmt.Errorf("icmp Message.Marshal: %v", err)
    }

    if n, err := c.IcmpConn.WriteTo(wb, c.TargetAddr); err != nil {
        return err
    } else if n != len(wb) {
        return fmt.Errorf("icmp Conn.WriteTo: did not send the whole message, %v", err)
    }

    return nil
}

func (c *Conn) NewMessage(id uint16, seq uint16) icmp.Message {
    wm := icmp.Message {
        Type: c.Proto.messageType,
        Code: 0,
        Body: &icmp.Echo {
            ID:     int(id),
            Seq:    int(seq),
            Data:   []byte("HELLO 1"),
        },
    }

    return wm
}
