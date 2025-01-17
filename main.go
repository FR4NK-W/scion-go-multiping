package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
	"golang.org/x/sync/errgroup"
)

func main() {
	Log.Debug("Go multiping")

	dia := addr.MustIAFrom(addr.ISD(71), addr.AS(559))
	dhost := net.UDPAddr{IP: net.ParseIP("10.10.0.1"), Port: 30041}
	remote := snet.UDPAddr{IA: dia, Host: &dhost}
	destIAs := []snet.UDPAddr{remote, {IA: addr.MustIAFrom(addr.ISD(71), addr.AS(8589934666)), Host: &dhost}}

	hc, err := initHostContext()
	if err != nil {
		Log.Debug("Failed init: ", err)
		os.Exit(1)
	}
	args := os.Args
	ipDestinations := []string{}

	remotesFile := "remotes.json"
	remotesEnv := os.Getenv("REMOTES_FILE")
	if remotesEnv != "" {
		remotesFile = remotesEnv
	}

	// Check if remotesFile exist
	if _, err := os.Stat(remotesFile); os.IsNotExist(err) {
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
	} else {
		remotes, err := parseRemotesJSON(remotesFile)
		if err != nil {
			Log.Error("Error parsing remotes file: ", err)
			os.Exit(1)
		}

		var destinationIAs []snet.UDPAddr
		for _, dest := range remotes.SCIONDestinations {
			dAddr, err := addr.ParseAddr(dest.Address)
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
			Log.Info("Added SCION destination: ", dest.Address, " for ", dest.Name)
		}

		for _, dest := range remotes.IPDestinations {
			ipDestinations = append(ipDestinations, dest.Address)
			Log.Info("Added IP destination: ", dest.Address, " for ", dest.Name)
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

	// Ping IP destinations
	go pingIPDestinations(prober, ipDestinations)

	Log.Info("Starting cron to write daily databases...")
	go dailyDatabaseUpdate(prober)

	Log.Info("Gathering results...")

	// Create a channel to receive OS signals
	signalChannel := make(chan os.Signal, 1)

	// Notify the channel for specific signals
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	// Create a channel to signal the program exit
	done := make(chan bool)

	// Goroutine to handle signals
	go func() {
		sig := <-signalChannel
		fmt.Printf("Received signal: %s\n", sig)
		err := prober.Exporter.Close()
		if err != nil {
			Log.Error("Failed to close database connection ", err)
		}
		done <- true
	}()

	fmt.Println("Press Ctrl+C to exit...")

	// Wait for a signal to be received
	<-done
}

func dailyDatabaseUpdate(prober *PathProber) {
	// Calculate the time until 12 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.FixedZone("UTC", 0)).Add(24 * time.Hour)
	durationUntilNextRun := time.Until(nextRun)

	// Wait until 12 AM
	Log.Infof("Waiting %s until the first run at 12 AM...\n", durationUntilNextRun)
	time.Sleep(durationUntilNextRun)

	// Start ticker to run the job every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		// Perform the task at 12 AM
		changeDailyDatabase(prober)

		// Wait for the next tick
		<-ticker.C
	}
}

func changeDailyDatabase(prober *PathProber) {
	err := prober.Exporter.InitDaily()
	if err != nil {
		Log.Error("Failed to change database connection to new file ", err)
		os.Exit(1)
	}

	Log.Info("Changed database connection to new file")

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

func pingIPDestinations(prober *PathProber, destinations []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var g errgroup.Group

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	hc := host()
	localIp := net.UDPAddr{IP: getSaddr(hc.hostInLocalAS), Port: 0}.IP.String()

	// We can only use one pinger here otherwise the raw sockets confuse their packets somehow?
	p, err := NewPinger(1)
	if err != nil {
		return err
	}

	for range ticker.C {
		for _, dest := range destinations {
			dest := dest // Capture range variable

			g.Go(func() error {
				select {
				case <-ctx.Done():
					return nil
				default:
					t := time.Now()
					var u IpUpdate
					pinger := p // pingers[dest]
					successChan := make(chan bool)
					timeChan := time.After(1 * time.Second)
					err := pinger.Send(dest, func(ipUpdate IpUpdate) {
						Log.Debug("Received IP Update ", u)
						u = ipUpdate
						successChan <- true
					})
					success := false
					if err != nil {
						Log.Error("Failed to send ping to remote ", dest)
					} else {
						select {
						case <-successChan:
							success = true
							break
						case <-timeChan:
							success = false
						}
					}

					diff := time.Since(t)
					// This is probably
					if err == nil && diff < 1 {
						Log.Debug("Skipping local ping result, probably the same host")
					} else {
						result := IPPingResult{
							DstAddr:  dest,
							SrcAddr:  localIp,
							Success:  err == nil && success,
							RTT:      float64(diff.Milliseconds()),
							PingTime: time.Now().UTC(),
						}

						err = prober.Exporter.WriteIPPingResult(result)
						if err != nil {
							Log.Debugf("Failed to write ping result: %v", err)
						}
					}

					return nil

				}
			})
		}
		g.Wait()
	}
	return nil
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
