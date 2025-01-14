package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
	"github.com/scionproto/scion/pkg/sock/reliable"
	"golang.org/x/sync/errgroup"
)

// Remember path states for the pinging module to know which paths to ping, i.e. the "PATH_STATE_PING" paths.
const (
	PATH_STATE_PING    = iota // Use for the current ping interval
	PATH_STATE_IDLE           // Not probed at all, we only that it is there
	PATH_STATE_PROBED         // Probed, but not selected for pinging. We know its latency.
	PATH_STATE_TIMEOUT        // This one timeouted, don't use it for a while
	PATH_STATE_DOWN           // Got an SCMP error, ignore it for now
)

// The result of probing a destination, containing the status of all paths to that destination.
type DestinationProbeResult struct {
	Paths []PathStatus
}

// The aggregated result of a probe over all paths to all destinations
type PathProbeResult struct {
	Destinations map[string]*DestinationProbeResult
}

// Used to store the status of a path to a destination, including the latency and the path itself.
// Based on this, the prober can select the proper paths for the pinging module
type PathStatus struct {
	State       int
	Path        *snet.Path
	Fingerprint string
	Latency     int64
}

// Represents a destination to probe, containing the remote address and the status of all paths to that destination.
type PingDestination struct {
	// Lock per entry, since map itself is concurrent-safe
	sync.Mutex
	PathStates []PathStatus
	RemoteAddr snet.UDPAddr
}

type PathProber struct {
	hostContext     *hostContext
	maxPathsToProbe int
	localIA         addr.IA
	localAddr       net.UDPAddr
	destinations    map[string]*PingDestination
}

// NewPathProber creates a new PathProber.
// The maxPathsToProbe parameter specifies the maximum number of paths to probe for each destination, to avoid probing dozens of paths.
func NewPathProber(localIA addr.IA, localAddr net.UDPAddr, maxPathsToProbe int) *PathProber {
	return &PathProber{
		destinations:    make(map[string]*PingDestination),
		maxPathsToProbe: maxPathsToProbe,
		localIA:         localIA,
		localAddr:       localAddr,
	}
}

// Inits the prober and does a path lookup to all destinations.
// TODO: Parallelize this
func (pb *PathProber) InitAndLookup() error {
	hc, err := initHostContext()
	if err != nil {
		return err
	}
	pb.hostContext = &hc

	var eg errgroup.Group
	for destStr, dest := range pb.destinations {
		eg.Go(func() error {
			fmt.Println("Querying paths to destination ", destStr)
			paths, err := hc.queryPaths(context.Background(), dest.RemoteAddr.IA)
			// TODO: Error handling
			if err != nil {
				fmt.Println("Error querying paths to destination ", destStr, ":", err)
				return err
			}

			fmt.Println("Found ", len(paths), " paths to destination ", destStr)

			for _, path := range paths {
				dest.PathStates = append(dest.PathStates, PathStatus{
					State:   PATH_STATE_IDLE,
					Path:    &path,
					Latency: 0,
					// Fingerprint: // TODO: Calculate fingerprint
				})
			}
			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		fmt.Println("Warning: Not all destinations were probed successfully")
		fmt.Println(err)
	}

	return nil
}

// Initially set all destinations to probe, needs to be done before InitAndLookup
func (pb *PathProber) SetDestinations(destinations []snet.UDPAddr) {
	for _, dest := range destinations {
		pb.destinations[dest.String()] = &PingDestination{
			RemoteAddr: dest,
			PathStates: make([]PathStatus, 0),
		}
	}
}

// Probe all paths to a given destination, returning the results.
func (pb *PathProber) Probe(destIsdAS string) (*DestinationProbeResult, error) {

	dest, ok := pb.destinations[destIsdAS]
	if !ok {
		return nil, fmt.Errorf("destination %s not found", destIsdAS)
	}

	result := &DestinationProbeResult{
		Paths: make([]PathStatus, min(pb.maxPathsToProbe, len(dest.PathStates))),
	}

	var eg errgroup.Group
	for i, pathStatus := range dest.PathStates {
		if i >= pb.maxPathsToProbe {
			break
		}
		eg.Go(func() error {
			// TODO: Reuse pingers here, this is not optimal to create a new pinger for each path
			// But I haven't found a way to reuse them yet
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
			saddr := dest.RemoteAddr.Copy()
			udpAddr := saddr.Host

			conn, port, err := svc.Register(ctx, saddr.IA, udpAddr, addr.SvcNone)
			if err != nil {
				return err
			}
			defer conn.Close()
			udpAddr.Port = int(port)

			p := pinger{
				timeout:       time.Second,
				pld:           make([]byte, 8),
				id:            id,
				conn:          conn,
				local:         &snet.UDPAddr{IA: saddr.IA, Host: udpAddr},
				replies:       replies,
				errHandler:    nil,
				updateHandler: nil,
			}
			rAddr := dest.RemoteAddr.Copy()
			rAddr.Path = (*pathStatus.Path).Dataplane()

			var update Update
			p.updateHandler = func(u Update) {
				fmt.Println("Got update ", u)
				update = u
			}
			// TODO: Stats here?
			_, err = p.SinglePing(ctx, rAddr)
			if err != nil {
				return err
			}
			rtt := update.RTT.Microseconds()
			fmt.Println("Probed path ", pathStatus.Path, " to destination ", destIsdAS, " with latency ", rtt)

			pathStatus.Latency = rtt
			result.Paths = append(result.Paths, PathStatus{
				State:   PATH_STATE_PROBED,
				Path:    pathStatus.Path,
				Latency: rtt,
				// Fingerprint: // TODO: Calculate fingerprint
			})
			return nil
		})
	}

	err := eg.Wait()

	return result, err
}

// Iterate over all destinations and probe all paths to each destination in parallel.
func (pb *PathProber) ProbeAll() (*PathProbeResult, error) {
	var eg errgroup.Group
	result := &PathProbeResult{
		Destinations: make(map[string]*DestinationProbeResult),
	}
	for _, dest := range pb.destinations {
		eg.Go(func() error {
			destAddrStr := dest.RemoteAddr.String()
			probeResult, err := pb.Probe(destAddrStr)
			if err != nil {
				return err
			}
			result.Destinations[destAddrStr] = probeResult
			return nil
		})
	}
	err := eg.Wait()
	return result, err
}

// Return the pathset to a given destination that should be used for pinging,
// i.e. [lowest latency path, shortest path in number of hops, most disjoint path(s)]
func (pb *PathProber) GetPathsForPing(destIsdAS string) ([]PathStatus, error) {
	return nil, nil
}
