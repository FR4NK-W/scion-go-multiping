package main

import (
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
)

func main() {
	Log.Debug("Go multiping")

	dia := addr.MustIAFrom(addr.ISD(71), addr.AS(559))
	dhost := net.UDPAddr{IP: net.ParseIP("10.10.0.1"), Port: 30041}
	remote := snet.UDPAddr{IA: dia, Host: &dhost}
	destIAs := []snet.UDPAddr{remote, snet.UDPAddr{IA: addr.MustIAFrom(addr.ISD(71), addr.AS(8589934666)), Host: &dhost}}

	hc, err := initHostContext()
	if err != nil {
		Log.Debug("Failed init: ", err)
		os.Exit(1)
	}
	args := os.Args
	if len(args) < 2 {
		Log.Warn("Run with arguments: ./bin/scion-go-multiping \"dest-ia-addr-1 dest-ia-2 ... dest-ia-addr-n\"")
		Log.Warn("Running with default values.")
		//os.Exit(0)
	} else {
		Log.Info("Received args: ", args)
		var destinationIAs []snet.UDPAddr
		for _, destAddr := range strings.Split(args[1], " ") {
			dAddr, err := addr.ParseAddr(destAddr)
			if err != nil {
				Log.Info("Invalid destination: ", dAddr, " error: ", err)
				os.Exit(1)
			}
			if dAddr.IA == hc.ia {
				Log.Debug("Not probing local AS: ", dAddr.IA)
				continue
			}
			destinationIAs = append(destinationIAs, snet.UDPAddr{IA: dAddr.IA, Host: &net.UDPAddr{
				IP:   dAddr.Host.IP().AsSlice(),
				Port: 30041,
			}})
		}
		destIAs = destinationIAs
	}

	// Path prober, e.g. probe up to 100 paths to each destination and ping up to 3 every second
	prober := NewPathProber(100, 3)
	prober.SetDestinations(destIAs)

	err = prober.InitAndLookup(hc)
	if err != nil {
		Log.Error("Error initializing and looking up paths:", err)
		os.Exit(1)
		return
	}
	Log.Info("Starting prober...")
	// Initial probing
	_, err = prober.ProbeAll()
	// TODO: Error handling?
	if err != nil {
		Log.Error("Error probing paths:", err)
	}
	err = prober.UpdatePathsToPing()
	if err != nil {
		Log.Error("Error updating paths to ping:", err)
	}

	Log.Info("Finished updating paths")

	Log.Debug("Selected paths for destination")
	for dest, paths := range pingPathSets.Paths {
		Log.Debug("Destination:", dest)
		for _, path := range paths {
			Log.Debug("Path:", path)
		}
	}

	// Sample usage, might be put into some other function or loop
	fullProbeTicker := time.NewTicker(60 * time.Second)
	go func() {
		for range fullProbeTicker.C {
			_, err := prober.ProbeAll()
			// TODO: Error handling?
			if err != nil {
				Log.Error("Error probing paths:", err)
				continue
			}

			Log.Info("Done probing all paths")
			// TODO: Write to SQLite here
			/*for dest, destResult := range results.Destinations {
				Log.Debug("Destination:", dest)
				for _, pathStatus := range destResult.Paths {
					Log.Debug("Path:", pathStatus.Path)
					Log.Debug("State:", pathStatus.State)
					Log.Debug("RTT:", pathStatus.RTT)
				}
			}*/
		}
	}()
	defer fullProbeTicker.Stop()
	Log.Info("Started full probe ticker")

	bestProbeTicker := time.NewTicker(1 * time.Second)
	go func() {
		for range bestProbeTicker.C {
			_, err := prober.ProbeBest()
			if err != nil {
				Log.Error("Error probing paths:", err)
				continue
			}
		}
	}()
	defer bestProbeTicker.Stop()
	Log.Info("Started best probe ticker")
	Log.Info("Gathering results...")
	// TODO: wait for ctrl +c or service interrupt

	time.Sleep(24 * time.Hour)
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

func getSaddr(dest net.IP) net.IP {
	udpAddr := net.UDPAddr{IP: dest, Port: 443}
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
	PathDown
	SCMPUnknown
)

type Stats struct {
	Sent     int
	Received int
	AvgRTT   float64
}
