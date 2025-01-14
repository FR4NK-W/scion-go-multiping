package main

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
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
	Destinations map[string]DestinationProbeResult
}

// Used to store the status of a path to a destination, including the latency and the path itself.
// Based on this, the prober can select the proper paths for the pinging module
type PathStatus struct {
	State       int
	Path        *snet.Path
	Fingerprint string
	Latency     float64
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

	for destStr, dest := range pb.destinations {
		paths, err := hc.queryPaths(context.Background(), dest.RemoteAddr.IA)
		// TODO: Error handling
		if err != nil {
			fmt.Println("Error querying paths to destination ", destStr, ":", err)
			continue
		}

		for _, path := range paths {
			dest.PathStates = append(dest.PathStates, PathStatus{
				State:   PATH_STATE_IDLE,
				Path:    &path,
				Latency: 0,
				// Fingerprint: // TODO: Calculate fingerprint
			})
		}
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

	return nil, nil
}

// Iterate over all destinations and probe all paths to each destination in parallel.
func (pb *PathProber) ProbeAll() (*PathProbeResult, error) {
	return nil, nil
}

// Return the pathset to a given destination that should be used for pinging,
// i.e. [lowest latency path, shortest path in number of hops, most disjoint path(s)]
func (pb *PathProber) GetPathsForPing(destIsdAS string) ([]PathStatus, error) {
	return nil, nil
}
