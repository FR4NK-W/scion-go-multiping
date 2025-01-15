package main

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
)

func main() {
	fmt.Println("Go multiping")

	dia := addr.MustIAFrom(addr.ISD(71), addr.AS(559))
	dhost := net.UDPAddr{IP: net.ParseIP("10.10.0.1"), Port: 30041}
	remote := snet.UDPAddr{IA: dia, Host: &dhost}
	destIAs := []snet.UDPAddr{remote, snet.UDPAddr{IA: addr.MustIAFrom(addr.ISD(71), addr.AS(8589934666)), Host: &dhost}}
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Run with arguments: ./bin/scion-go-multiping \"dest-ia-addr-1 dest-ia-2 ... dest-ia-addr-n\"")
		fmt.Println("Running with default values.")
		//os.Exit(0)
	} else {
		fmt.Println(args)
		var destinationIAs []snet.UDPAddr
		for _, destAddr := range strings.Split(args[1], " ") {
			dAddr, err := addr.ParseAddr(destAddr)
			if err != nil {
				fmt.Println("Invalid destination: ", dAddr, " error: ", err)
				os.Exit(1)
			}
			destinationIAs = append(destinationIAs, snet.UDPAddr{IA: dAddr.IA, Host: &net.UDPAddr{
				IP:   dAddr.Host.IP().AsSlice(),
				Port: 30041,
			}})
		}
		destIAs = destinationIAs
	}

	// Path prober, e.g. probe up to 10 paths to each destination and ping up to 3 every second
	prober := NewPathProber(10, 3)
	prober.SetDestinations(destIAs)

	err := prober.InitAndLookup()
	if err != nil {
		fmt.Println("Error initializing and looking up paths:", err)
		os.Exit(1)
		return
	}

	// Initial probing
	_, err = prober.ProbeAll()
	// TODO: Error handling?
	if err != nil {
		fmt.Println("Error probing paths:", err)
	}
	err = prober.UpdatePathsToPing()
	if err != nil {
		fmt.Println("Error updating paths to ping:", err)
	}

	fmt.Println("Selected paths for destination")
	for dest, paths := range pingPathSets.Paths {
		fmt.Println("Destination:", dest)
		for _, path := range paths {
			fmt.Println("Path:", path)
		}
	}

	// Sample usage, might be put into some other function or loop
	fullProbeTicker := time.NewTicker(5 * time.Second)
	go func() {
		for range fullProbeTicker.C {
			_, err := prober.ProbeAll()
			// TODO: Error handling?
			if err != nil {
				fmt.Println("Error probing paths:", err)
				continue
			}

			// TODO: Write to SQLite here
			/*for dest, destResult := range results.Destinations {
				fmt.Println("Destination:", dest)
				for _, pathStatus := range destResult.Paths {
					fmt.Println("Path:", pathStatus.Path)
					fmt.Println("State:", pathStatus.State)
					fmt.Println("Latency:", pathStatus.Latency)
				}
			}*/
		}
	}()
	defer fullProbeTicker.Stop()

	bestProbeTicker := time.NewTicker(1 * time.Second)
	go func() {
		for range bestProbeTicker.C {
			_, err := prober.ProbeBest()
			if err != nil {
				fmt.Println("Error probing paths:", err)
				continue
			}
		}
	}()
	defer bestProbeTicker.Stop()
	time.Sleep(20 * time.Second)
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
)

type Stats struct {
	Sent       int
	Received   int
	AvgLatency float64
}
