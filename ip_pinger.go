package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

type IpPinger struct {
	sync.Mutex
	id               uint16
	conn             *net.IPConn
	pld              []byte
	sentSequence     int
	receivedSequence int
	replies          <-chan IpPingReply
	updateHandlers   map[int]func(IpUpdate)
}

type IpPingReply struct {
	Received time.Time
	Source   net.Addr
	Size     int
	SeqNum   int
	RTT      time.Duration
	Error    error
}

type IpUpdate struct {
	RTT      time.Duration
	Sequence int
	Size     int
	Source   net.Addr
}

func NewPinger(id uint16) (*IpPinger, error) {
	conn, err := net.ListenIP("ip4:icmp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw socket: %w", err)
	}

	p := &IpPinger{
		id:             id,
		conn:           conn,
		pld:            make([]byte, 8),
		updateHandlers: make(map[int]func(IpUpdate)),
	}
	go p.receiveLoop(context.Background())
	return p, nil
}

func (p *IpPinger) Send(dest string, updateHandler func(IpUpdate)) error {
	p.Lock()
	sequence := p.sentSequence + 1
	p.updateHandlers[sequence] = updateHandler
	p.sentSequence = sequence
	p.Unlock()

	binary.BigEndian.PutUint64(p.pld, uint64(time.Now().UnixNano()))
	icmpMessage := buildICMPMessage(p.id, uint16(sequence), p.pld)
	dstAddr, err := net.ResolveIPAddr("ip4", dest)
	if err != nil {
		return fmt.Errorf("failed to resolve destination address: %w", err)
	}

	fmt.Println("Sending sequence ", sequence)
	if _, err := p.conn.WriteTo(icmpMessage, dstAddr); err != nil {
		return fmt.Errorf("failed to send ICMP packet: %w", err)
	}

	return nil
}

func (p *IpPinger) receiveLoop(ctx context.Context) {
	buffer := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, addr, err := p.conn.ReadFrom(buffer)
			if err != nil {
				Log.Error("Failed to get ping reply: ", err)
				continue
			}

			rtt := time.Duration(0)
			seqNum := 0
			if n >= 16 { // Minimal ICMP echo reply size
				seqNum = int(binary.BigEndian.Uint16(buffer[6:8]))
				// sentTime := int64(binary.BigEndian.Uint16(buffer[14:16]))
				// rtt = time.Since(time.Unix(0, sentTime))
			}

			if handler, ok := p.updateHandlers[seqNum]; ok {
				p.Lock()
				handler(IpUpdate{
					Source:   addr,
					Size:     n,
					RTT:      rtt,
					Sequence: seqNum,
				})
				delete(p.updateHandlers, seqNum)
				fmt.Println(len(p.updateHandlers))
				p.Unlock()
			}
		}
	}
}

func buildICMPMessage(id, seq uint16, payload []byte) []byte {
	header := make([]byte, 8+len(payload))
	header[0] = 8 // Type: Echo Request
	header[1] = 0 // Code
	binary.BigEndian.PutUint16(header[4:6], id)
	binary.BigEndian.PutUint16(header[6:8], seq)
	copy(header[8:], payload)

	checksum := calculateChecksum(header)
	binary.BigEndian.PutUint16(header[2:4], checksum)
	return header
}

func calculateChecksum(data []byte) uint16 {
	sum := 0
	for i := 0; i < len(data)-1; i += 2 {
		sum += int(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if len(data)%2 == 1 {
		sum += int(data[len(data)-1]) << 8
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	return ^uint16(sum)
}

/*func main() {
	pinger, err := NewPinger(12345, 2*time.Second, 1*time.Second)
	if err != nil {
		fmt.Printf("Error creating pinger: %v\n", err)
		return
	}

	updateHandler := func(update Update) {
		fmt.Printf("Reply from %v: bytes=%d seq=%d time=%v\n",
			update.Source, update.Size, update.Sequence, update.RTT)
	}

	destination := "8.8.8.8"
	for i := 0; i < 4; i++ {
		if err := pinger.Send(destination, updateHandler); err != nil {
			fmt.Printf("Error sending ping: %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}
}*/
