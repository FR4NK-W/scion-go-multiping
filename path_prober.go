package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
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
	PATH_STATE_PROBED         // Probed, but not selected for pinging. We know its RTT.
	PATH_STATE_TIMEOUT        // This one timeouted, don't use it for a while
	PATH_STATE_DOWN           // Got an SCMP error, ignore it for now
	PATH_STATE_UNKNOWN        // Something went wrong here, maybe not use the path
)

// The result of probing a destination, containing the status of all paths to that destination.
type DestinationProbeResult struct {
	Paths []PathStatus
}

// The aggregated result of a probe over all paths to all destinations
type PathProbeResult struct {
	Destinations map[string]*DestinationProbeResult
}

// Used to store the status of a path to a destination, including the rtt and the path itself.
// Based on this, the prober can select the proper paths for the pinging module
type PathStatus struct {
	State       int
	Path        snet.Path
	Fingerprint string
	RTT         int64
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
	maxPathsToProbe int // Max paths to probe every minute, current: 10
	maxPathsToPing  int // Max paths to ping every second, current: 3
	localIA         addr.IA
	localAddr       net.UDPAddr
	destinations    map[string]*PingDestination
	Exporter        DataExporter
	pingers         map[string]*pinger
}

// NewPathProber creates a new PathProber.
// The maxPathsToProbe parameter specifies the maximum number of paths to probe for each destination, to avoid probing dozens of paths.
func NewPathProber(maxPathsToProbe int, maxPathsToPing int) *PathProber {
	return &PathProber{
		destinations:    make(map[string]*PingDestination, maxPathsToProbe),
		maxPathsToProbe: maxPathsToProbe,
		maxPathsToPing:  maxPathsToPing,
		Exporter:        NewSQLiteExporter(),
		pingers:         make(map[string]*pinger),
	}
}

// Inits the prober and does a path lookup to all destinations.
// TODO: Parallelize this
func (pb *PathProber) InitAndLookup(hc hostContext) error {
	pb.hostContext = &hc
	pb.localAddr = net.UDPAddr{IP: getSaddr(hc.hostInLocalAS), Port: 0}
	pb.localIA = hc.ia
	fmt.Println("Local IA: ", pb.localIA)
	fmt.Println("Local Addr: ", pb.localAddr)

	err := pb.Exporter.InitDaily()
	if err != nil {
		return err
	}

	var eg errgroup.Group
	for destStr, dest := range pb.destinations {
		eg.Go(func() error {
			Log.Debug("Querying paths to destination ", destStr)
			paths, err := hc.queryPaths(context.Background(), dest.RemoteAddr.IA)
			// TODO: Error handling
			if err != nil {
				Log.Error("Error querying paths to destination ", destStr, ":", err)
				return err
			}

			Log.Debug("Found ", len(paths), " paths to destination ", destStr)

			for _, path := range paths {
				dest.PathStates = append(dest.PathStates, PathStatus{
					State:       PATH_STATE_IDLE,
					Path:        path,
					RTT:         0,
					Fingerprint: calculateFingerprint(path),
				})
			}
			return nil
		})

		ctx := context.TODO()
		replies := make(chan reply, 50)
		id := snet.RandomSCMPIdentifer()
		handler := scmpHandler{
			id:      id,
			replies: replies,
		}
		udpAddr := pb.localAddr

		conn, port, err := newSCIONConn(ctx, handler, pb.localIA, udpAddr)
		if err != nil {
			return err
		}
		udpAddr.Port = int(port)

		p := pinger{
			timeout:        time.Second,
			pld:            make([]byte, 8),
			id:             id,
			conn:           conn,
			local:          &snet.UDPAddr{IA: pb.localIA, Host: &udpAddr},
			replies:        replies,
			errHandler:     nil,
			updateHandler:  nil,
			updateHandlers: make(map[int]func(Update)),
		}
		p.runReceiveLoop()
		pb.pingers[destStr] = &p

	}

	err = eg.Wait()
	if err != nil {
		Log.Debug("Warning: Not all destinations were probed successfully: ", err)
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
		Paths: make([]PathStatus, 0),
	}

	if len(dest.PathStates) == 0 {
		Log.Error("No paths to probe for ", destIsdAS)
	}

	var eg errgroup.Group
	lookuptime := time.Now().UTC()
	for i, pathStatus := range dest.PathStates {
		if i >= pb.maxPathsToProbe {
			break
		}
		eg.Go(func() error {
			// TODO: Reuse pingers here, this is not optimal to create a new pinger for each path
			// But I haven't found a way to reuse them yet

			pinger := pb.pingers[destIsdAS]

			rAddr := dest.RemoteAddr.Copy()
			rAddr.Path = pathStatus.Path.Dataplane()
			rAddr.NextHop = pathStatus.Path.UnderlayNextHop()

			var update Update
			successChan := make(chan bool)
			timeChan := time.After(700 * time.Millisecond)
			// Log.Debug("Sending ping to ", rAddr, " via ", pathStatus.Fingerprint)
			err := pinger.Send(rAddr, func(u Update) {
				// Log.Debug("Got update ", u, " from ", rAddr, " via ", pathStatus.Fingerprint)
				update = u
				successChan <- true
			})

			// TODO: Error Handling, is this a path timeout or path down?
			if err != nil {
				fmt.Println("Error sending ping to ", rAddr, " via ", pathStatus.Fingerprint, "err: ", err)
				return err
			}

			success := false

			select {
			case <-timeChan:
				Log.Debug("Timeout for ", rAddr, " via ", pathStatus.Fingerprint)
				break
			case <-successChan:
				success = true
				break
			}

			if success {
				rtt := update.RTT.Milliseconds()

				state := PATH_STATE_PROBED
				switch update.State {
				case PathDown:
					state = PATH_STATE_DOWN
				case SCMPUnknown:
					state = PATH_STATE_UNKNOWN
				}

				pathStatus.RTT = rtt
				result.Paths = append(result.Paths, PathStatus{
					State:       state,
					Path:        pathStatus.Path,
					RTT:         rtt,
					Fingerprint: pathStatus.Fingerprint,
				})
			} else {
				result.Paths = append(result.Paths, PathStatus{
					State:       PATH_STATE_TIMEOUT,
					Path:        pathStatus.Path,
					Fingerprint: pathStatus.Fingerprint,
				})
			}

			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		Log.Debug("Not all probes to dest ", destIsdAS, " successfull")
	}

	successCount := 0
	minRTT := int64(10000000000)
	maxRTT := int64(0)

	minHops := 100000
	maxHops := 0

	var pathStrings []string
	var pathFingerprints []string
	for _, path := range result.Paths {

		if path.RTT > 0 {
			successCount++

			if path.RTT < minRTT {
				minRTT = path.RTT
			}

			if path.RTT > maxRTT {
				maxRTT = path.RTT
			}

			pathLen := len(path.Path.Metadata().Interfaces)

			if pathLen < minHops {
				minHops = pathLen
			}

			if pathLen > maxHops {
				maxHops = pathLen
			}

		}
		interfacesString := ""
		for i, iface := range path.Path.Metadata().Interfaces {
			if i == 0 {
				interfacesString = iface.String()
				continue
			}
			interfacesString += "->" + iface.String()
		}
		pathStrings = append(pathStrings, interfacesString)
		pathFingerprints = append(pathFingerprints, path.Fingerprint)
	}

	ps := PathStatistics{
		SrcSCIONAddr:   fmt.Sprintf("%s,%s", pb.localIA.String(), pb.localAddr.String()),
		DstSCIONAddr:   destIsdAS,
		Paths:          strings.Join(pathStrings, ","),
		Fingerprints:   strings.Join(pathFingerprints, ","),
		Success:        successCount > 0,
		MinRTT:         float64(minRTT),
		MaxRTT:         float64(maxRTT),
		MinHops:        minHops,
		MaxHops:        maxHops,
		LookupTime:     lookuptime,
		ActivePaths:    successCount,
		ProbedPaths:    len(result.Paths),
		AvailablePaths: len(pb.destinations[destIsdAS].PathStates),
	}

	err = pb.Exporter.WritePathStatistic(ps)
	if err != nil {
		Log.Error("Error writing path statistic for ", destIsdAS, ":", err)
		return nil, err
	}

	return result, err
}

func (pb *PathProber) UpdatePathList(destStr string, dest *PingDestination) error {
	Log.Debug("Querying paths to destination ", destStr)
	hc := host()
	paths, err := hc.queryPaths(context.Background(), dest.RemoteAddr.IA)
	// TODO: Error handling
	if err != nil {
		Log.Error("Error querying paths to destination ", destStr, ":", err)
		return err
	}

	Log.Debug("Found ", len(paths), " paths to destination ", destStr)

	for _, path := range paths {
		fp := calculateFingerprint(path)
		foundIndex := -1
		for i, pathStatus := range dest.PathStates {
			if pathStatus.Fingerprint == fp {
				foundIndex = i
				// TODO: Update path status here? We already know the path
			}
		}

		// We need to update the path with a new entry
		if foundIndex > 0 {
			Log.Debug("Updating path ", path, " for ", destStr)
			dest.PathStates[foundIndex].Path = path
		} else {
			dest.PathStates = append(dest.PathStates, PathStatus{
				State:       PATH_STATE_IDLE,
				Path:        path,
				RTT:         0,
				Fingerprint: fp,
			})
		}
	}
	return nil
}

// Iterate over all destinations and probe all paths to each destination in parallel.
func (pb *PathProber) ProbeAll() (*PathProbeResult, error) {
	var eg errgroup.Group
	result := &PathProbeResult{
		Destinations: make(map[string]*DestinationProbeResult),
	}
	for destStr, dest := range pb.destinations {
		eg.Go(func() error {

			err := pb.UpdatePathList(destStr, dest)
			if err != nil {
				Log.Error("Error updating path list for ", destStr, ":", err)
			}

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
		Paths: make([]PathStatus, 0),
	}

	pingPathSets.Lock()
	pingPathSetsPaths := pingPathSets.Paths[dest.RemoteAddr.String()]
	pingPathSets.Unlock()
	var eg errgroup.Group

	for _, path := range pingPathSetsPaths {
		eg.Go(func() error {

			pinger := pb.pingers[destIsdAS]

			rAddr := dest.RemoteAddr.Copy()
			rAddr.Path = path.Dataplane()
			rAddr.NextHop = path.UnderlayNextHop()

			var update Update
			successChan := make(chan bool)
			timeChan := time.After(700 * time.Millisecond)
			Log.Debug("Sending bestProbe to ", rAddr, " via ", path)
			err := pinger.Send(rAddr, func(u Update) {
				Log.Debug("Got update for bestprobe ", u, " from ", rAddr, " via ", path)
				update = u
				successChan <- true
			})

			success := false

			select {
			case <-timeChan:
				Log.Debug("Timeout for ", rAddr, " via ", path)
				break
			case <-successChan:
				success = true
				break
			}

			// TODO: Error Handling, is this a path timeout or path down?
			if err != nil {
				return err
			}

			if success {
				rtt := update.RTT.Milliseconds()
				state := PATH_STATE_PROBED
				switch update.State {
				case PathDown:
					state = PATH_STATE_DOWN
				case SCMPUnknown:
					state = PATH_STATE_UNKNOWN
				}

				result.Paths = append(result.Paths, PathStatus{
					State:       state,
					Path:        path,
					RTT:         rtt,
					Fingerprint: calculateFingerprint(path),
				})
			} else {
				result.Paths = append(result.Paths, PathStatus{
					State:       PATH_STATE_TIMEOUT,
					Path:        path,
					Fingerprint: calculateFingerprint(path),
				})
			}

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
	t := time.Now()
	Log.Info("Probing best run... ")
	timeout := time.After(2 * time.Second)
	for _, dest := range pb.destinations {
		eg.Go(func() error {
			destAddrStr := dest.RemoteAddr.String()
			pingtime := time.Now().UTC()
			probeResult, err := pb.ProbeDestBest(destAddrStr)
			if err != nil {
				return err
			}
			minRTT := int64(1000000)
			successCount := 0

			Log.Debug("Probed ", destAddrStr, " got entries ", len(probeResult.Paths))
			var minRTTPathFingerPrint string
			for _, path := range probeResult.Paths {
				Log.Debug("Path1 ", path.Path, " has RTT ", path.RTT)
				if path.RTT > 0 {
					successCount++
					if path.RTT < minRTT {
						minRTT = path.RTT
						minRTTPathFingerPrint = path.Fingerprint
					}
				}
			}

			pr := PingResult{
				SrcSCIONAddr:    fmt.Sprintf("%s,%s", pb.localIA.String(), pb.localAddr.String()),
				DstSCIONAddr:    destAddrStr,
				Success:         successCount > 0,
				RTT:             float64(minRTT),
				Fingerprint:     minRTTPathFingerPrint,
				PingTime:        pingtime,
				SuccessfulPings: successCount,
				MaxPings:        len(probeResult.Paths),
			}
			err = pb.Exporter.WritePingResult(pr)
			if err != nil {
				Log.Error("Error writing ping result for ", destAddrStr, ":", err)
				return err
			}

			result.Destinations[destAddrStr] = probeResult
			return nil
		})
	}

	// We seem to have some weird behaviour here, where the errgroup doesn't return the error
	// the tool stucks after a few hours, the last thing logged is Log.Info("Probing best run... ")
	// and then everything stops
	// So maybe in the send itself something blocks, so hopefully this resolves it
	var err error
	doneChan := make(chan bool)
	go func() {
		err = eg.Wait()
		doneChan <- true

	}()
	select {
	case <-timeout:
		err = errors.New("probing best run timed out in 2s")
		Log.Error("Probing best run timed out in 2s")
	case <-doneChan:
		// err := eg.Wait()
		diff := time.Since(t)
		Log.Info("Probing best run took ", diff)
	}

	return result, err
}

// Return the pathset to a given destination that should be used for pinging,
// i.e. [lowest rtt path, shortest path in number of hops, most disjoint path(s)]
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
		if len(paths) == 0 {
			Log.Error("No paths to ping selected for ", destStr)
			continue
		}
		pingPathSets.Paths[destStr] = paths
	}

	return nil
}

/*
*
Path Selection Algorithm: (every 60 seconds or when at least 2 pings fail to a destination)
  - Input: NetworkState filled with rtt, number of hops, etc, Output: List of up to 3 paths
  - 1. Ignore all paths that have state "down" or "timeout"
  - 2. If number of paths  <3 the choose all paths
  - 3. Select shortest path in number of hops
  - 4. Select lowest rtt path
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
	// 4. Select lowest rtt path
	shortPaths := shortestAndLowestRTTPath(activePaths)

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

func shortestAndLowestRTTPath(paths []PathStatus) []PathStatus {
	minHops := 100000
	var shortestPath PathStatus

	minRTT := int64(100000)
	var lowestRTTPath PathStatus

	for _, path := range paths {
		metaData := path.Path.Metadata()

		if len(metaData.Interfaces) < minHops {
			minHops = len(metaData.Interfaces) / 2
			shortestPath = path
		}

		if path.RTT < minRTT {
			minRTT = path.RTT
			lowestRTTPath = path
		}
	}

	if shortestPath.Fingerprint == lowestRTTPath.Fingerprint {
		return []PathStatus{shortestPath}
	}

	return []PathStatus{shortestPath, lowestRTTPath}
}

// calculateFingerprint generates a unique fingerprint for a path by hashing its interfaces
func calculateFingerprint(path snet.Path) string {
	return snet.Fingerprint(path).String()
}
