package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/log"
	"github.com/scionproto/scion/pkg/private/common"
	"github.com/scionproto/scion/pkg/private/serrors"
	"github.com/scionproto/scion/pkg/snet"
	"github.com/scionproto/scion/pkg/snet/path"
	"github.com/scionproto/scion/pkg/sock/reliable"
	"github.com/scionproto/scion/private/topology/underlay"
)

func main() {
	fmt.Println("Go multiping")

	saddr := net.UDPAddr{IP: getSaddr(), Port: 0}
	fmt.Println("saddr", saddr)
	sia := addr.MustIAFrom(addr.ISD(64), addr.AS(8589934601))
	dia := addr.MustIAFrom(addr.ISD(71), addr.AS(559))
	dhost := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	remote := snet.UDPAddr{IA: dia, Host: &dhost}
	destIAs := []snet.UDPAddr{remote, snet.UDPAddr{IA: addr.MustIAFrom(addr.ISD(71), addr.AS(8589934666)), Host: &dhost}}
	args := os.Args
	if len(args) < 3 {
		fmt.Println("Run with arguments: ./bin/scion-go-multiping local-ia \"dest-ia-1 dest-ia-2 ... dest-ia-n\"")
		fmt.Println("Running with default values.")
		//os.Exit(0)
	} else {
		localIA := args[1]
		sia, _ = addr.ParseIA(localIA)
		var destinationIAs []snet.UDPAddr
		for _, destIA := range strings.Split(args[2], " ") {
			ia, _ := addr.ParseIA(destIA)
			destinationIAs = append(destinationIAs, snet.UDPAddr{IA: ia, Host: &dhost})
		}
		destIAs = destinationIAs
	}

	for _, r := range destIAs {
		// Short interval
		go runPing(sia, saddr, r, 10*time.Second)
	}

	// Long interval
	runPing(sia, saddr, remote, 60*time.Second)

	// Path prober, e.g. probe up to 10 paths to each destination
	prober := NewPathProber(sia, saddr, 10)
	prober.SetDestinations(destIAs)

	// Sample usage, might be put into some other function or loop
	probeTicker := time.NewTicker(60 * time.Second)
	go func() {
		for range probeTicker.C {
			results, err := prober.ProbeAll()
			// TODO: Error handling?
			if err != nil {
				fmt.Println("Error probing paths:", err)
				continue
			}

			// TODO: Write to SQLite here
			for dest, destResult := range results.Destinations {
				fmt.Println("Destination:", dest)
				for _, pathStatus := range destResult.Paths {
					fmt.Println("Path:", pathStatus.Path)
					fmt.Println("State:", pathStatus.State)
					fmt.Println("Latency:", pathStatus.Latency)
				}
			}
		}
	}()
	defer probeTicker.Stop()

}

func runPing(sia addr.IA, saddr net.UDPAddr, r snet.UDPAddr, interval time.Duration) {
	ctx := context.TODO()
	replies := make(chan reply, 50)
	id := snet.RandomSCMPIdentifer()
	dispSockerPath := getDispatcherPath()
	svc := snet.DefaultPacketDispatcherService{
		Dispatcher: reliable.NewDispatcher(dispSockerPath),
		SCMPHandler: scmpHandler{
			id:      id,
			replies: replies,
		},
	}

	conn, port, err := svc.Register(ctx, sia, &saddr, addr.SvcNone)
	if err != nil {
		return
	}
	saddr.Port = int(port)

	p := pinger{
		interval:      interval,
		timeout:       time.Second,
		pld:           make([]byte, 8),
		id:            id,
		conn:          conn,
		local:         &snet.UDPAddr{IA: sia, Host: &saddr},
		replies:       replies,
		errHandler:    nil,
		updateHandler: nil,
	}
	p.Ping(ctx, &r)
}

func getDispatcherPath() string {
	pathCandidates := []string{
		"/var/run/dispatcher/default.sock",
		"/run/shm/dispatcher/default.sock",
	}
	for _, candidate := range pathCandidates {
		if fInfo, err := os.Stat(candidate); err == nil && fInfo.Mode()&os.ModeSocket != 0 {
			return candidate
		}
	}
	return ""
}

func getSaddr() net.IP {
	ip := net.ParseIP("129.132.19.216")
	udpAddr := net.UDPAddr{IP: ip, Port: 443}
	var err error
	var conn *net.UDPConn
	if conn, err = net.DialUDP(udpAddr.Network(), nil, &udpAddr); err == nil {
		return net.ParseIP(netip.MustParseAddrPort(conn.LocalAddr().String()).Addr().String())
	}
	return nil
}

type Update struct {
	Size     int
	Source   snet.SCIONAddress
	Sequence int
	RTT      time.Duration
	State    State
}

type State int

const (
	Success State = iota
	AfterTimeout
	Duplicate
)

type Stats struct {
	Sent       int
	Received   int
	AvgLatency float64
}

type pinger struct {
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
}

func (p *pinger) Ping(ctx context.Context, remote *snet.UDPAddr) (Stats, error) {
	p.sentSequence, p.receivedSequence = -1, -1
	send := time.NewTicker(p.interval)
	defer send.Stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errSend := make(chan error, 1)

	go func() {
		defer func() {
			log.Debug("Draining")
		}()
		p.drain(ctx)
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// sender

		for {
			if err := p.send(remote); err != nil {
				errSend <- serrors.WrapStr("sending", err)
				continue
			}
			select {
			case <-send.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return p.stats, nil
		case err := <-errSend:
			return p.stats, err
		case reply := <-p.replies:
			if reply.Error != nil {
				if p.errHandler != nil {
					p.errHandler(reply.Error)
				}
				continue
			}
			p.receive(reply)
		}
	}
}

func (p *pinger) send(remote *snet.UDPAddr) error {
	sequence := p.sentSequence + 1

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

	p.sentSequence = sequence
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
