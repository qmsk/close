package udp

// syscall.Socket/Sendto -based implementation. Supports send with arbitrary source

import (
    "fmt"
    "net"
    "syscall"
)

type SockSyscall struct {
    ipAddr      net.IPAddr
    udpAddr     net.UDPAddr

    // XXX: cleanup..
    fd          int
    sockaddr    syscall.Sockaddr

    stats       *SockStats
}

func (self *SockSyscall) init(dstAddr string) error {
    self.stats = &SockStats{}

    // resolve
    if udpAddr, err := net.ResolveUDPAddr("udp", dstAddr); err != nil {
        return fmt.Errorf("Resolve UDP %v: %v", dstAddr, err)
    } else {
        self.ipAddr = net.IPAddr{IP: udpAddr.IP, Zone: udpAddr.Zone}
        self.udpAddr = *udpAddr
    }

    // setup
    if ip4 := self.udpAddr.IP.To4(); ip4 != nil {
        sockaddr := &syscall.SockaddrInet4{}

        for i, b := range self.ipAddr.IP.To4() {
            sockaddr.Addr[i] = b
        }

        self.sockaddr = sockaddr

        if fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK, syscall.IPPROTO_RAW); err != nil {
            return err
        } else {
            self.fd = fd
        }

    } else if ip6 := self.udpAddr.IP.To16(); ip6 != nil {
        return fmt.Errorf("TODO: IPv6")
    } else {
        return fmt.Errorf("Invalid IP family")
    }

    return nil
}

// probe the source address the kernel would select for our destination
func (self *SockSyscall) probeSource() (*net.UDPAddr, error) {
    if udpConn, err := net.DialUDP("udp", nil, &self.udpAddr); err != nil {
        return nil, err
    } else {
        return udpConn.LocalAddr().(*net.UDPAddr), nil
    }
}

func (self *SockSyscall) send(packet Packet) error {
    packetBytes, err := packet.PackIP()
    if err != nil {
        return err
    }

    // send
    if err := syscall.Sendto(self.fd, packetBytes, 0, self.sockaddr); err == nil {
        self.stats.Packets++
        self.stats.Bytes += uint(len(packetBytes)) // XXX: doesn't Sendto() return...?
    } else {
        // TODO: check for EAGAIN -> self.stats.Dropped++
        self.stats.Errors++
    }

    return nil
}

func (self *SockSyscall) useStats(stats *SockStats) {
    self.stats = stats
}
