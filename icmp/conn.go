package icmp

import (
    "fmt"
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv4"
    "golang.org/x/net/ipv6"
    "net"
    "time"
)

// ICMP Echo Request/Response info
type Ping struct {
    Time    time.Time
    IP      net.IP
    ID      uint16
    Seq     uint16
    Data    []byte
}

type Conn struct {
    targetAddr  *net.UDPAddr
    icmpConn    *icmp.PacketConn
    proto       proto

    // ICMP ID, chosen by kernel
    id          int
}

type proto struct {
    listenStr   string
    resolveStr  string
    ianaProto   int
    messageType icmp.Type

    decodeAddr  func(net.Addr) (net.IP, int, error)
}

func decodeAddr(netAddr net.Addr) (net.IP, int, error) {
    switch addr := netAddr.(type) {
    case *net.UDPAddr:
        return addr.IP, addr.Port, nil
    default:
        return nil, 0, fmt.Errorf("Unkonwn net.Addr: %T %#v", addr, addr)
    }
}

// IANA ICMP v4 protocol is 1
// IANA ICMP v6 protocol is 58
var (
    protocols = map[string]proto {
        "ipv4": proto {
            resolveStr:  "ip4",
            listenStr:   "udp4",
            ianaProto:   1,
            messageType: ipv4.ICMPTypeEcho,
        },
        "ipv6": proto {
            resolveStr: "ip6",
            listenStr:  "udp6",
            ianaProto:  58,
            messageType: ipv6.ICMPTypeEchoRequest,
        },
    }
)

// NOTE: for unprivileged ICMP sockets, the given id will always be ignored, since the kernel assigns an ID..
func NewConn(protoName string, target string, id int) (*Conn, error) {
    if target == "" {
        return nil, fmt.Errorf("No target given")
    }

    c := &Conn {

    }

    if proto, exists := protocols[protoName]; !exists {
        return nil, fmt.Errorf("protocol not registered: %v", protoName)
    } else {
        c.proto = proto
    }

    if ipAddr, err := net.ResolveIPAddr(c.proto.resolveStr, target); err != nil {
        return nil, fmt.Errorf("net.ResolveIPAddr %v:%v: %v", c.proto.resolveStr, target, err)
    } else {
        // unprivileged icmp mode uses SOCK_DGRAM
        c.targetAddr = &net.UDPAddr{IP: ipAddr.IP, Zone: ipAddr.Zone}
    }

    if icmpConn, err := icmp.ListenPacket(c.proto.listenStr, ""); err != nil {
        return nil, fmt.Errorf("icmp.ListenPacket %v: %v", c.proto.listenStr, err)
    } else {
        c.icmpConn = icmpConn
    }

    // store local address
    if _, id, err := decodeAddr(c.icmpConn.LocalAddr()); err != nil {
        return nil, fmt.Errorf("Unkonwn icmpConn.LocalAddr(): %v", err)
    } else {
        c.id = id
    }

    return c, nil
}

func (c *Conn) String() string {
    return fmt.Sprintf("%v", c.targetAddr.IP)
}

func (c *Conn) ID() int {
    return c.id
}

func (c *Conn) Write(wm icmp.Message) (error) {
    wb, err := wm.Marshal(nil)
    if err != nil {
        return fmt.Errorf("icmp Message.Marshal: %v", err)
    }

    if n, err := c.icmpConn.WriteTo(wb, c.targetAddr); err != nil {
        return err
    } else if n != len(wb) {
        return fmt.Errorf("icmp Conn.WriteTo: did not send the whole message, %v", err)
    }

    return nil
}

// Send ICMP Echo Request using given parameters.
//
// Updates ping.ID and ping.Time when sending.
func (c *Conn) Send(ping *Ping) error {
    wm := icmp.Message {
        Type: c.proto.messageType,
        Code: 0,
        Body: &icmp.Echo {
            ID:     int(ping.ID),
            Seq:    int(ping.Seq),
            Data:   ping.Data,
        },
    }

    wb, err := wm.Marshal(nil)
    if err != nil {
        return fmt.Errorf("icmp Message.Marshal: %v", err)
    }

    // send
    ping.Time = time.Now()
    ping.IP = c.targetAddr.IP
    ping.ID = uint16(c.id)

    if n, err := c.icmpConn.WriteTo(wb, c.targetAddr); err != nil {
        return err
    } else if n != len(wb) {
        return fmt.Errorf("icmp Conn.WriteTo: did not send the whole message, %v", err)
    }

    return nil
}

// Receive ICMP Echo Response.
func (c *Conn) Recv() (ping Ping, err error) {
    buf := make([]byte, 1500)

    if readSize, readAddr, err := c.icmpConn.ReadFrom(buf); err != nil {
        // TODO: return nil if the connection is closed, report other errors?
        return ping, err

    } else if ip, _, err := decodeAddr(readAddr); err != nil {
        return ping, err

    } else {
        buf = buf[:readSize]
        ping.IP = ip
    }

    ping.Time = time.Now()

    if icmpMessage, err := icmp.ParseMessage(c.proto.ianaProto, buf); err != nil {
        return ping, err
    } else if icmpEcho, ok := icmpMessage.Body.(*icmp.Echo); ok {
        ping.ID = uint16(icmpEcho.ID)
        ping.Seq = uint16(icmpEcho.Seq)
    } else {
        return ping, fmt.Errorf("Unknown message: %v\n", err)
    }

    return ping, nil
}

func (c *Conn) Close() error {
    return c.icmpConn.Close()
}
