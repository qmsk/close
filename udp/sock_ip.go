package udp

// net.IPConn based socket. Supports send with arbitrary source

import (
    "fmt"
    "golang.org/x/net/ipv4"
    "net"
)

type SockIP struct {
    ipAddr      net.IPAddr
    udpAddr     net.UDPAddr

    ipConn      *net.IPConn
    rawConn     *ipv4.RawConn

    stats       SockStats
}

func (self *SockIP) init(dstAddr string) error {
    // resolve
    if udpAddr, err := net.ResolveUDPAddr("udp", dstAddr); err != nil {
        return fmt.Errorf("Resolve UDP %v: %v", dstAddr, err)
    } else {
        self.ipAddr = net.IPAddr{IP: udpAddr.IP, Zone: udpAddr.Zone}
        self.udpAddr = *udpAddr
    }

    if ipConn, err := net.ListenIP("ip:udp", nil); err != nil {
        return fmt.Errorf("ListenIP: %v", err)
    } else {
        self.ipConn = ipConn
    }

    // setup
    if ip4 := self.udpAddr.IP.To4(); ip4 != nil {
        // XXX: only used for the side-effect of setsockopt IP_HDRINCL
        if rawConn, err := ipv4.NewRawConn(self.ipConn); err != nil {
            return fmt.Errorf("NewRawConn: %v", err)
        } else {
            self.rawConn = rawConn
        }

    } else if ip6 := self.udpAddr.IP.To16(); ip6 != nil {
        return fmt.Errorf("TODO: IPv6")
    } else {
        return fmt.Errorf("Invalid IP family")
    }

    return nil
}

// probe the source address the kernel would select for our destination
func (self *SockIP) probeSource() (*net.UDPAddr, error) {
    if udpConn, err := net.DialUDP("udp", nil, &self.udpAddr); err != nil {
        return nil, err
    } else {
        return udpConn.LocalAddr().(*net.UDPAddr), nil
    }
}

// serialize and send from Packet
func (self *SockIP) send(packet Packet) error {
    packetBytes, err := packet.PackIP()
    if err != nil {
        return err
    }

    // send
    if send, err := self.ipConn.WriteToIP(packetBytes, &self.ipAddr); err != nil {
        self.stats.Errors++
    } else {
        self.stats.Packets++
        self.stats.Bytes += uint(send)
    }

    return nil
}

func (self *SockIP) takeStats() SockStats {
    stats := self.stats
    self.stats = SockStats{}

    return stats
}
