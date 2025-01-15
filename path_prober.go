package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
	"github.com/scionproto/scion/pkg/sock/reliable"
	"golang.org/x/sync/errgroup"
)

type PingPathSets struct {
	// In case a ping module updates the paths to ping e.g. when a path is down
	sync.Mutex
	// Map of destination addresses to the paths to ping
	Paths map[string][]snet.Path
}

var pingPathSets PingPathSets

// Remember path states for the pinging module to know which paths to ping, i.e. the "PATH_STATE_PING" paths.
const (
	PATH_STATE_PING    = iota // Used for the current ping interval
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
	Path        snet.Path
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
	exporter        DataExporter
}

// NewPathProber creates a new PathProber.
// The maxPathsToProbe parameter specifies the maximum number of paths to probe for each destination, to avoid probing dozens of paths.
func NewPathProber(maxPathsToProbe int) *PathProber {
	return &PathProber{
		destinations:    make(map[string]*PingDestination, maxPathsToProbe),
		maxPathsToProbe: maxPathsToProbe,
		exporter:        NewSQLiteExporter(),
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
	pb.localAddr = net.UDPAddr{IP: getSaddr(hc.hostInLocalAS), Port: 0}
	pb.localIA = hc.ia

	err = pb.exporter.Init()
	if err != nil {
		return err
	}

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
					State:       PATH_STATE_IDLE,
					Path:        path,
					Latency:     0,
					Fingerprint: calculateFingerprint(path),
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
			udpAddr := pb.localAddr

			conn, port, err := svc.Register(ctx, pb.localIA, &udpAddr, addr.SvcNone)
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
				local:         &snet.UDPAddr{IA: pb.localIA, Host: &udpAddr},
				replies:       replies,
				errHandler:    nil,
				updateHandler: nil,
			}
			rAddr := dest.RemoteAddr.Copy()
			rAddr.Path = pathStatus.Path.Dataplane()
			rAddr.NextHop = pathStatus.Path.UnderlayNextHop()

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

			pathStatus.Latency = rtt
			result.Paths = append(result.Paths, PathStatus{
				State:       PATH_STATE_PROBED,
				Path:        pathStatus.Path,
				Latency:     rtt,
				Fingerprint: pathStatus.Fingerprint,
			})
			return nil
		})
	}

	successCount := 0
	minLatency := int64(10000000)
	maxLatency := int64(0)

	minHops := 100000
	maxHops := 0

	for _, path := range result.Paths {
		if path.Latency > 0 {
			successCount++

			if path.Latency < minLatency {
				minLatency = path.Latency
			}

			if path.Latency > maxLatency {
				maxLatency = path.Latency
			}

			pathLen := len(path.Path.Metadata().Interfaces) / 2

			if pathLen < minHops {
				minHops = pathLen
			}

			if pathLen > maxHops {
				maxHops = pathLen
			}

		}
	}

	err := eg.Wait()
	if err != nil {
		fmt.Println("Not all probes to dest ", destIsdAS, " successfull")
	}

	ps := PathStatistics{
		SrcSCIONAddr:   fmt.Sprintf("%s,%s", pb.localIA.String(), pb.localAddr.String()),
		DstSCIONAddr:   destIsdAS,
		Success:        successCount > 0,
		MinLatency:     float64(minLatency),
		MaxLatency:     float64(maxLatency),
		MinHops:        minHops,
		MaxHops:        maxHops,
		LookupTime:     time.Now(),
		ActivePaths:    successCount,
		ProbedPaths:    len(result.Paths),
		AvailablePaths: len(pb.destinations[destIsdAS].PathStates),
	}

	err = pb.exporter.WritePathStatistic(ps)
	if err != nil {
		return nil, err
	}

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

// Probe the selected paths from pingPathSets to a given destination, returning the results.
func (pb *PathProber) ProbeDestBest(destIsdAS string) (*DestinationProbeResult, error) {

	dest, ok := pb.destinations[destIsdAS]
	if !ok {
		return nil, fmt.Errorf("destination %s not found", destIsdAS)
	}

	result := &DestinationProbeResult{
		Paths: make([]PathStatus, min(pb.maxPathsToProbe, len(dest.PathStates))),
	}
	var eg errgroup.Group
	for _, path := range pingPathSets.Paths[dest.RemoteAddr.String()] {
		eg.Go(func() error {
			// TODO: Reuse pingers here
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
			udpAddr := pb.localAddr

			conn, port, err := svc.Register(ctx, pb.localIA, &udpAddr, addr.SvcNone)
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
				local:         &snet.UDPAddr{IA: pb.localIA, Host: &udpAddr},
				replies:       replies,
				errHandler:    nil,
				updateHandler: nil,
			}
			rAddr := dest.RemoteAddr.Copy()
			rAddr.Path = path.Dataplane()
			rAddr.NextHop = path.UnderlayNextHop()

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

			result.Paths = append(result.Paths, PathStatus{
				State:       PATH_STATE_PROBED,
				Path:        path,
				Latency:     rtt,
				Fingerprint: calculateFingerprint(path),
			})
			return nil
		})
	}

	err := eg.Wait()

	return result, err
}

// Iterate over all destinations and only probe the selected best paths from pingPathSets to each destination in parallel.
func (pb *PathProber) ProbeBest() (*PathProbeResult, error) {
	var eg errgroup.Group
	result := &PathProbeResult{
		Destinations: make(map[string]*DestinationProbeResult),
	}
	for _, dest := range pb.destinations {
		eg.Go(func() error {
			destAddrStr := dest.RemoteAddr.String()
			probeResult, err := pb.ProbeDestBest(destAddrStr)
			if err != nil {
				return err
			}
			minLatency := int64(1000000)
			successCount := 0

			fmt.Println("Probed ", destAddrStr)
			for _, path := range probeResult.Paths {
				fmt.Println("Path ", path.Path, " has latency ", path.Latency)
				if path.Latency > 0 {
					successCount++
					if path.Latency < minLatency {
						minLatency = path.Latency
					}
				}
			}

			pr := PingResult{
				SrcSCIONAddr:    fmt.Sprintf("%s,%s", pb.localIA.String(), pb.localAddr.String()),
				DstSCIONAddr:    destAddrStr,
				Success:         successCount > 0,
				Latency:         float64(minLatency),
				PingTime:        time.Now(),
				SuccessfulPings: successCount,
				MaxPings:        len(probeResult.Paths),
			}
			err = pb.exporter.WritePingResult(pr)
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

// Updates the variable that holds the paths to ping for each destination.
// Should be called after each ProbeAll call.
func (pb *PathProber) UpdatePathsToPing() error {
	pingPathSets.Lock()
	defer pingPathSets.Unlock()
	if pingPathSets.Paths == nil {
		pingPathSets.Paths = make(map[string][]snet.Path)
	}

	for destStr, dest := range pb.destinations {
		paths := pb.SelectOptimalPathsToPing(dest)
		pingPathSets.Paths[destStr] = paths
	}

	return nil
}

/*
*
Path Selection Algorithm: (every 60 seconds or when at least 2 pings fail to a destination)
  - Input: NetworkState filled with latency, number of hops, etc, Output: List of up to 3 paths
  - 1. Ignore all paths that have state "down" or "timeout"
  - 2. If number of paths  <3 the choose all paths
  - 3. Select shortest path in number of hops
  - 4. Select lowest latency path
  - 5. If those two result in the same path, select one highly disjoint path in addition to it
  - 6. Select the most disjoint / most diverse path with respect to the previously selected paths
*/
func (pb *PathProber) SelectOptimalPathsToPing(pingDestination *PingDestination) []snet.Path {

	// 1. Ignore all paths that have state "down" or "timeout"
	activePaths := make([]PathStatus, 0)
	for _, path := range pingDestination.PathStates {
		if path.State == PATH_STATE_DOWN || path.State == PATH_STATE_TIMEOUT {
			continue
		}
		activePaths = append(activePaths, path)
	}

	// 2. If number of paths  <3 the choose all paths
	if len(activePaths) <= 3 {
		paths := make([]snet.Path, 0)
		for _, path := range activePaths {
			paths = append(paths, path.Path)
		}
		return paths
	}

	// 3. Select shortest path in number of hops
	// 4. Select lowest latency path
	shortPaths := shortestAndLowestLatencyPath(activePaths)

	// 5 & 6: Select up to 2 most disjoint paths in addition to this
	pathSet := addMostDisjointPaths(shortPaths, activePaths)

	paths := make([]snet.Path, 0)
	for _, path := range pathSet {
		paths = append(paths, path.Path)
	}
	return paths

}

func addMostDisjointPaths(shortestPaths []PathStatus, allPaths []PathStatus) []PathStatus {
	// Create a map to track selected fingerprints for easier comparison
	selectedFingerprints := make(map[string]bool)
	for _, path := range shortestPaths {
		selectedFingerprints[path.Fingerprint] = true
	}

	// Helper function to calculate disjointness score
	calculateDisjointness := func(path1, path2 snet.Path) int {
		meta1 := path1.Metadata()
		meta2 := path2.Metadata()
		interfaceSet := make(map[string]bool)

		for _, iface := range meta1.Interfaces {
			interfaceSet[iface.String()] = true
		}

		disjointCount := 0
		for _, iface := range meta2.Interfaces {
			if !interfaceSet[iface.String()] {
				disjointCount++
			}
		}

		return disjointCount
	}

	// Sort remaining paths by their disjointness from the selected paths
	var candidatePaths []PathStatus
	for _, path := range allPaths {
		if !selectedFingerprints[path.Fingerprint] {
			candidatePaths = append(candidatePaths, path)
		}
	}

	// Rank candidates by disjointness score
	type disjointPath struct {
		path       PathStatus
		disjointed int
	}
	var rankedCandidates []disjointPath
	for _, candidate := range candidatePaths {
		disjointScore := 0
		for _, selected := range shortestPaths {
			disjointScore += calculateDisjointness(candidate.Path, selected.Path)
		}
		rankedCandidates = append(rankedCandidates, disjointPath{
			path:       candidate,
			disjointed: disjointScore,
		})
	}

	// Sort candidates by disjointness score (descending order)
	sort.Slice(rankedCandidates, func(i, j int) bool {
		return rankedCandidates[i].disjointed > rankedCandidates[j].disjointed
	})

	// Select top-ranked paths until we reach a total of 3 paths
	for i := 0; i < len(rankedCandidates) && len(shortestPaths) < 3; i++ {
		shortestPaths = append(shortestPaths, rankedCandidates[i].path)
	}

	return shortestPaths
}

func shortestAndLowestLatencyPath(paths []PathStatus) []PathStatus {
	minHops := 100000
	var shortestPath PathStatus

	minLatency := int64(100000)
	var lowestLatencyPath PathStatus

	for _, path := range paths {
		metaData := path.Path.Metadata()

		if len(metaData.Interfaces) < minHops {
			minHops = len(metaData.Interfaces) / 2
			shortestPath = path
		}

		if path.Latency < minLatency {
			minLatency = path.Latency
			lowestLatencyPath = path
		}
	}

	// TODO: We need fingerprints for this
	if shortestPath.Fingerprint == lowestLatencyPath.Fingerprint {
		return []PathStatus{shortestPath}
	}

	return []PathStatus{shortestPath, lowestLatencyPath}
}

// calculateFingerprint generates a unique fingerprint for a path by hashing its interfaces
func calculateFingerprint(path snet.Path) string {
	meta := path.Metadata()
	var interfaces []string

	// Collect interface identifiers
	for _, iface := range meta.Interfaces {
		interfaces = append(interfaces, iface.String())
	}

	// Sort interfaces to ensure consistent order for the fingerprint
	sort.Strings(interfaces)

	// Concatenate all interfaces into a single string
	concatenated := ""
	for _, iface := range interfaces {
		concatenated += iface
	}

	// Hash the concatenated string using SHA-256
	hash := sha256.Sum256([]byte(concatenated))

	// Convert the hash to a hexadecimal string and return as the fingerprint
	return hex.EncodeToString(hash[:])
}
