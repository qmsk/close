package ping

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"os"
	"log"
	"time"
	"net"
	"sync"
)

func getAddr(dst string) (net.Addr, error) {
	if ips, err := net.LookupIP(dst); err != nil {
		return nil, err
	} else {
		return &net.UDPAddr{IP: ips[0]}, err
	}
}

type Pinger struct {
	seq         int
	dst         net.Addr
	conn        *icmp.PacketConn
	RTT         chan  time.Duration
	seqM        *sync.Mutex
	startTimes  map[int]time.Time
}

func NewPinger(dst string) (*Pinger, error) {
	p := &Pinger {
		seq: 1,
	}

        conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Printf("Could not start listening: %s\n", err)
		return nil, err
	}
	p.conn = conn

	udpAddr, err := getAddr(dst)
	if err != nil {
		log.Printf("Could not resolve remote address: %s\n", err)
		return nil, err
	}

	p.dst = udpAddr
	p.RTT = make(chan time.Duration)
	p.seqM = &sync.Mutex {}
	p.startTimes = make(map[int]time.Time)

	go p.receiver()

	return p, nil
}

func (p *Pinger) Close() {
	p.conn.Close()
	close(p.RTT)
}

func (p *Pinger) Latency() {
	go p.ping()
}

func (p *Pinger) ping() {
	p.seqM.Lock()
	p.seq += 1
	seq := p.seq
	p.seqM.Unlock()

	wm := icmp.Message {
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo {
			ID: os.Getpid() & 0xffff, Seq: seq,
			Data: []byte("HELLO 1"),
		},
	}

	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Printf("Could not marshal ICMP message: %s\n", err)
	}

	p.startTimes[seq] = time.Now()

	n, err := p.conn.WriteTo(wb, p.dst)
	if n != len(wb) || err != nil {
		log.Printf("Could not send the packet: %s\n", err)
		delete(p.startTimes, seq)
	}
}

func (p *Pinger) receiver() {
	for {
		rb := make([]byte, 1500)
		n, _, err := p.conn.ReadFrom(rb)
		if err != nil {
			log.Printf("Could not read the ICMP reply: %s\n", err)
		}

		stop := time.Now()

		// IANA ICMP v4 protocol is 1
		rm, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			log.Printf("Could not parse the ICMP reply: %s\n", err)
			continue
		}

		if rm.Type == ipv4.ICMPTypeEchoReply {
			echo := rm.Body.(*icmp.Echo)
			if start, ok := p.startTimes[echo.Seq]; ok {
				p.RTT <- stop.Sub(start)
			}
		}
	}
}
