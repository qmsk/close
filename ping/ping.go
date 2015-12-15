package ping

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"os"
	"log"
	"time"
	"net"
)

func getAddr(dst string) (net.Addr, error) {
	if ips, err := net.LookupIP(dst); err != nil {
		return nil, err
	} else {
		return &net.UDPAddr{IP: ips[0]}, err
	}
}

type pingResult struct {
	Seq   int
	Stop  time.Time
}

type Pinger struct {
	seq         int
	DstName     string
	dst         net.Addr
	conn        *icmp.PacketConn
	RTT         chan  time.Duration
	senderC     chan  bool
	receiverC   chan  pingResult
	startTimes  map[int]time.Time
}

func NewPinger(dst string) (*Pinger, error) {
	p := &Pinger {
		seq: 1,
		DstName: dst,
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
	p.senderC = make(chan bool)
	p.receiverC = make(chan pingResult)
	p.startTimes = make(map[int]time.Time)

	go p.receiver()
	go p.manager()

	return p, nil
}

func (p *Pinger) Close() {
	p.conn.Close()
	close(p.RTT)
	close(p.senderC)
	close(p.receiverC)
}

func (p *Pinger) Latency() {
	p.senderC <- true
}

func (p *Pinger) manager() {
	for {
		select {
		case result, ok := <-p.receiverC:
			if !ok {
				break
			}
			if start, ok := p.startTimes[result.Seq]; ok {
				p.RTT <- result.Stop.Sub(start)
				delete(p.startTimes, result.Seq)
			}
		case <-p.senderC:
			p.ping()
//		case <-expiryTicker.C:
		}
	}
}

// Now this is called only from manager, so it's okay it's not thread safe
func (p *Pinger) ping() {
	p.seq += 1
	seq := p.seq

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
	// If the receiver channel was closed then trying to send on it will panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in Pinger.receiver()", r)
		}
	}()
	for {
		rb := make([]byte, 1500)
		n, _, err := p.conn.ReadFrom(rb)
		// TODO If the connection is closed quit the loop
		if err != nil {
			log.Printf("Could not read the ICMP reply: %s\n", err)
			continue
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
			p.receiverC <- pingResult { echo.Seq, stop, }
		}
	}
}
