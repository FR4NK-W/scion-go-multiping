package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/private/common"
	"github.com/scionproto/scion/pkg/private/serrors"
	"github.com/scionproto/scion/pkg/snet"
	"github.com/scionproto/scion/pkg/snet/path"
	"github.com/scionproto/scion/private/topology/underlay"
)

type pinger struct {
	sync.Mutex
	interval time.Duration
	timeout  time.Duration

	id            uint16
	conn          snet.PacketConn
	local         *snet.UDPAddr
	replies       <-chan reply
	errHandler    func(error)
	updateHandler func(Update)

	pld              []byte
	sentSequence     int
	receivedSequence int
	stats            Stats
	updateHandlers   map[int]func(Update)
}

// TODO: Context, cancellation
func (p *pinger) runReceiveLoop() {

	go p.drain(context.Background())

	go func() {
		for reply := range p.replies {
			if reply.Error != nil {
				if p.errHandler != nil {
					p.errHandler(reply.Error)
				}
			}
			p.receive(reply)
		}
	}()
}

func (p *pinger) Send(remote *snet.UDPAddr, updateHandler func(Update)) error {
	p.Lock()
	sequence := p.sentSequence + 1
	p.updateHandlers[sequence] = updateHandler
	p.sentSequence = sequence
	p.Unlock()

	binary.BigEndian.PutUint64(p.pld, uint64(time.Now().UnixNano()))
	pkt, err := packSCMPrequest(p.local, remote, snet.SCMPEchoRequest{
		Identifier: p.id,
		SeqNumber:  uint16(sequence),
		Payload:    p.pld,
	})
	if err != nil {
		return err
	}
	nextHop := remote.NextHop
	if nextHop == nil && p.local.IA.Equal(remote.IA) {
		nextHop = &net.UDPAddr{
			IP:   remote.Host.IP,
			Port: underlay.EndhostPort,
			Zone: remote.Host.Zone,
		}
	}

	if err := p.conn.WriteTo(pkt, nextHop); err != nil {
		return err
	}

	p.stats.Sent++

	return nil
}

func (p *pinger) receive(reply reply) {
	rtt := reply.Received.Sub(time.Unix(0, int64(binary.BigEndian.Uint64(reply.Reply.Payload))))
	var state State
	switch {
	case rtt > p.timeout:
		state = AfterTimeout
	case int(reply.Reply.SeqNumber) == p.receivedSequence:
		state = Duplicate
	case int(reply.Reply.SeqNumber) == p.receivedSequence+1:
		state = Success
		p.receivedSequence = int(reply.Reply.SeqNumber)
	default:
	}

	p.stats.Received++
	if p.updateHandler != nil {
		p.updateHandler(Update{
			RTT:      rtt,
			Sequence: int(reply.Reply.SeqNumber),
			Size:     reply.Size,
			Source:   reply.Source,
			State:    state,
		})
	}

	if handler, ok := p.updateHandlers[int(reply.Reply.SeqNumber)]; ok {
		p.Lock()
		handler(Update{
			RTT:      rtt,
			Sequence: int(reply.Reply.SeqNumber),
			Size:     reply.Size,
			Source:   reply.Source,
			State:    state,
		})
		delete(p.updateHandlers, int(reply.Reply.SeqNumber))
		p.Unlock()
	}
}

func (p *pinger) drain(ctx context.Context) {
	var last time.Time
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var pkt snet.Packet
			var ov net.UDPAddr
			if err := p.conn.ReadFrom(&pkt, &ov); err != nil && p.errHandler != nil {
				if now := time.Now(); now.Sub(last) > time.Second {
					p.errHandler(serrors.WrapStr("straggler packet", err))
					last = now
				}
			}
		}
	}
}

func packSCMPrequest(local, remote *snet.UDPAddr, req snet.SCMPEchoRequest) (*snet.Packet, error) {
	_, isEmpty := remote.Path.(path.Empty)
	if isEmpty && !local.IA.Equal(remote.IA) {
		return nil, serrors.New("no path to remote IA", "local", local.IA, "remote", remote.IA)
	}
	remoteHostIP, ok := netip.AddrFromSlice(remote.Host.IP)
	if !ok {
		return nil, serrors.New("invalid remote IP", "ip", remote.Host.IP)
	}
	localHostIP, ok := netip.AddrFromSlice(local.Host.IP)
	if !ok {
		return nil, serrors.New("invalid local IP", "ip", local.Host.IP)
	}

	pkt := &snet.Packet{
		PacketInfo: snet.PacketInfo{
			Destination: snet.SCIONAddress{
				IA:   remote.IA,
				Host: addr.HostIP(remoteHostIP),
			},
			Source: snet.SCIONAddress{
				IA:   local.IA,
				Host: addr.HostIP(localHostIP),
			},
			Path:    remote.Path,
			Payload: req,
		},
	}
	return pkt, nil
}

type reply struct {
	Received time.Time
	Source   snet.SCIONAddress
	Size     int
	Reply    snet.SCMPEchoReply
	Error    error
}

type scmpHandler struct {
	id      uint16
	replies chan<- reply
}

func (h scmpHandler) Handle(pkt *snet.Packet) error {
	echo, err := h.handle(pkt)
	if err != nil {
		fmt.Println("Error handling packet ", err)
	}
	h.replies <- reply{
		Received: time.Now(),
		Source:   pkt.Source,
		Size:     len(pkt.Bytes),
		Reply:    echo,
		Error:    err,
	}
	return nil
}

func (h scmpHandler) handle(pkt *snet.Packet) (snet.SCMPEchoReply, error) {
	if pkt.Payload == nil {
		return snet.SCMPEchoReply{}, serrors.New("no timing payload found")
	}
	switch s := pkt.Payload.(type) {
	case snet.SCMPEchoReply:
		r := pkt.Payload.(snet.SCMPEchoReply)
		if r.Identifier != h.id {
			return snet.SCMPEchoReply{}, serrors.New("wrong SCMP ID",
				"expected", h.id, "actual", r.Identifier)
		}
		return r, nil
	case snet.SCMPExternalInterfaceDown:
		return snet.SCMPEchoReply{}, serrors.New("external interface down",
			"isd_as", s.IA, "interface", s.Interface)
	default:
	}
	return snet.SCMPEchoReply{}, serrors.New("not an SCMPEchoReply",
		"type", common.TypeOf(pkt.Payload))
}
